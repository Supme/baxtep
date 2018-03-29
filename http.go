package baxtep

import (
	"html/template"
	"net/http"
	"strings"
	"context"
	"time"
	"io"
	"fmt"
)

type Handler struct {
	Config *HandlerConfig
}

type HandlerConfig struct {
	Pattern             string
	Baxter              *Baxtep
	ContextName			string
	RedirectAfterLogin  *string
	RedirectAfterLogout *string
	SessionDuration     time.Duration
	ConfirmRegistration func(http.ResponseWriter, *http.Request, string)
	LogWriter			io.Writer
	tmpl                *template.Template
}

func NewHandler(config *HandlerConfig) *Handler {
	handler := Handler{Config: config}
	return &handler
}

func (uh *Handler) SetCustomTemplate(tmpl *template.Template) {
	uh.Config.tmpl = tmpl
}

func (uh *Handler) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(strings.ToLower(r.URL.String()), uh.Config.Pattern) {
			// Если отработал встроенный метод выходим
			if uh.HandlerFunc(w, r) {
				return
			}
		}
		uh.checkHandler(h).ServeHTTP(w, r)
	})
}

func (uh *Handler) HandlerFunc(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := r.URL.Query()["login"]; ok {
		uh.login(w, r)
		return true
	} else if _, ok := r.URL.Query()["logout"]; ok {
		uh.logout(w, r)
		return true
	} else if _, ok := r.URL.Query()["cameout"]; ok {
		uh.cameout(w, r)
		return true
	} else if _, ok := r.URL.Query()["registration"]; ok {
		uh.registration(w, r)
		return true
	} else {
		uh.Base(w, r)
		return true
	}
	return false
}

// ToDo captcha
func (uh *Handler) registration(w http.ResponseWriter, r *http.Request) {
	// has confirmation link?
	if len(r.URL.Query()["registration"]) > 0 && r.URL.Query()["registration"][0] != "" {
		uh.confirmation(w, r)
		return
	}
	if r.Method == "POST" {
		if r.FormValue("name") == "" {
			http.Error(w, "Blank name", http.StatusOK)
			return
		}
		if r.FormValue("email") == "" {
			http.Error(w, "Blank email", http.StatusOK)
			return
		}
		if r.FormValue("password") == "" {
			http.Error(w, "Blank password", http.StatusOK)
			return
		}
		if r.FormValue("password") == "" {
			http.Error(w, "Blank retry password", http.StatusOK)
			return
		}
		if r.FormValue("password") != r.FormValue("retry-password") {
			http.Error(w, "Password does not match", http.StatusOK)
			return
		}
		user := uh.Config.Baxter.AddNewUser(r.FormValue("name"), r.FormValue("email"))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	err := uh.checkTemplate()
	if err != nil {
		uh.logPrintf( "Registration template error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = uh.Config.tmpl.ExecuteTemplate(w, "_userregistration", nil)
	if err != nil {
		uh.logPrintf("Registration template execute error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (uh *Handler) confirmation(w http.ResponseWriter, r *http.Request) {
	u, err := uh.Config.Baxter.ConfirmRegistration(r.URL.Query()["registration"][0])
	if err == ErrUserNotFound {
		http.Error(w, "Bad confirmation link", http.StatusForbidden)
		return
	}
	if err != nil {
		uh.logPrintf( "Confirmation error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sessionID, err := u.SetNewSessionID()
	if err != nil {
		uh.logPrintf("Confirmation SetNewSession error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	cookie := http.Cookie{
		Path:     "/",
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(uh.Config.SessionDuration),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	w.Header().Set("Content-Type", "text/html")
	err = uh.checkTemplate()
	if err != nil {
		uh.logPrintf( "Confirmation template error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = uh.Config.tmpl.ExecuteTemplate(w, "_userconfirmation", u)
	if err != nil {
		uh.logPrintf("Confirmation template execute error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}


// ToDo captcha
func (uh *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.FormValue("email") == "" {
			http.Error(w, "Blank email", http.StatusForbidden)
			return
		}
		if r.FormValue("password") == "" {
			http.Error(w, "Blank password", http.StatusForbidden)
			return
		}
		u, err := uh.Config.Baxter.GetUserByEmailPassword(r.FormValue("email"), r.FormValue("password"))
		if err != nil {
			switch err {
			case ErrUserWithEmailNotFound, ErrUserBadPassword:
				 http.Error(w, "Wrong email or password", http.StatusForbidden)
			default:
				uh.logPrintf("Login GetUserByEmailPassword error: %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		sessionID, err := u.SetNewSessionID()
		if err != nil {
			uh.logPrintf("Login SetNewSession error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		cookie := http.Cookie{
			Path:     "/",
			Name:     "session_id",
			Value:    sessionID,
			Expires:  time.Now().Add(uh.Config.SessionDuration),
			HttpOnly: true,
		}
		http.SetCookie(w, &cookie)
		if uh.Config.RedirectAfterLogin != nil {
			http.Redirect(w, r, *uh.Config.RedirectAfterLogin, http.StatusFound)
			return
		}
		http.Redirect(w, r, "?base", http.StatusFound)
		return
	}

	// else GET method
	w.Header().Set("Content-Type", "text/html")
	err := uh.checkTemplate()
	if err != nil {
		uh.logPrintf("Login template error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	userdata := uh.getUserData(w, r)
	err = uh.Config.tmpl.ExecuteTemplate(w, "_userlogin", userdata)
	if err != nil {
		uh.logPrintf("Login template execute error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (uh *Handler) logout(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires: time.Unix(0, 0),
		HttpOnly: true,
	}
	http.SetCookie(w, c)
	if uh.Config.RedirectAfterLogout != nil {
		http.Redirect(w, r, *uh.Config.RedirectAfterLogout, http.StatusFound)
		return
	}
	http.Redirect(w, r, "?comeout", http.StatusFound)
}

// cameout page after exit ???
func (uh *Handler) cameout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := uh.checkTemplate()
	if err != nil {
		uh.logPrintf("Cameout template error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = uh.Config.tmpl.ExecuteTemplate(w, "_usercameout", nil)
	if err != nil {
		uh.logPrintf("Cameout template execute error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (uh *Handler) Base(w http.ResponseWriter, r *http.Request) {
	userdata := uh.getUserData(w, r)
	w.Header().Set("Content-Type", "text/html")
	err := uh.checkTemplate()
	if err != nil {
		uh.logPrintf("Base template error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = uh.Config.tmpl.ExecuteTemplate(w, "_userbase", userdata)
	if err != nil {
		uh.logPrint("Base template execute error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (uh *Handler) getUserData(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	data := map[string]interface{}{}
	ctx := uh.check(w, r)
	r = r.WithContext(ctx)
	if ok := r.Context().Value(uh.Config.ContextName); ok == nil {
		return data
	}
	user := r.Context().Value(uh.Config.ContextName).(User)
	data["_User"] = user
	params, err := user.GetParams()
	if err != nil {
		uh.logPrintf("Error in GetUserData: $%s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return data
	}
	data["_UserParams"] = params
	return data
}

func (uh *Handler) checkHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := uh.check(w, r)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func  (uh *Handler) check(w http.ResponseWriter, r *http.Request) context.Context {
	var user User
	ctx := context.Background()
	sessionID, err := r.Cookie("session_id")
	if err != http.ErrNoCookie {
		if err != nil {
			uh.logPrintf("User check cookie error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return ctx
		}

		user, err = uh.Config.Baxter.GetUserBySessionID(sessionID.Value)
		if err != nil && err != ErrUserSessionNotFound {
			uh.logPrint( err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return ctx
		}

		err = user.CheckSessionID(uh.Config.SessionDuration)
		if err == ErrUserSessionExpired || err == ErrUserNotFound {
			return ctx
		}
		if err != nil {
			uh.logPrint( err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return ctx
		}
		ctx = context.WithValue(r.Context(), uh.Config.ContextName, user)
	}
	return ctx
}

func (uh *Handler) checkTemplate() error {
	if uh.Config.tmpl == nil {
		// use default template
		var err error
		uh.Config.tmpl, err = template.New("_UserHandler").Parse(defaultTemplate)
		if err != nil {
			return err
		}
	}
	return nil
}


func (uh *Handler) logPrint(s ...interface{}) {
	if uh.Config.LogWriter != nil {
		fmt.Fprint(uh.Config.LogWriter, s...)
	}
}

func (uh *Handler) logPrintf(f string, s ...interface{}) {
	if uh.Config.LogWriter != nil {
		fmt.Fprintf(uh.Config.LogWriter, f, s...)
	}
}
package main

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"encoding/base64"
	"net/http"
	"github.com/supme/baxtep"
	"log"
	"time"
	"github.com/cznic/ql"
	"bytes"
)

var db *sql.DB

var (
	// driverName = "mysql"
	// dataSource = "baxtep:baxtep@tcp(localhost:3306)/baxtep?parseTime=true"
	driverName = "ql-mem"
	dataSource = "ql.db"
)

func main() {
	var err error
	ql.RegisterDriver()
	ql.RegisterMemDriver()
	db, err = sql.Open(driverName,dataSource)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Start webserver on %s\n", "8080")
	err = Start(":8080")
	if err != nil {
		panic(err)
	}
}

func Start(listenAddres string) error {
	baxta := baxtep.NewBaxtep(db, driverName, "user")
	err := baxta.InitDB()
	if err != nil {
		panic(err)
	}

	var logWriter LogWriter
	userContextName := "BAXTER"

	baxtepHandler := baxtep.NewHandler(&baxtep.HandlerConfig{
		Pattern:             "/user",
		ContextName: userContextName,
		Baxter:              baxta,
		RedirectAfterLogin:  nil,
		RedirectAfterLogout: nil,
		SessionDuration: time.Hour * 24,
		LogWriter: logWriter,
	})

	err = baxta.CheckExistUserName("user")
	if err != nil && err != baxtep.ErrUserNameExist {
		panic(err)
	} else {
		user, confirm, err := baxta.AddNewUser("user", "user@domain.tld")
		if err != nil {
			log.Print(err)
		}
		fmt.Printf("User email '%s', confirmation link '%s'\n", user.Email, confirm)
		user.SetNewPassword("userpass")
		user.AddParams(
			map[string]string{"for delete 1": "test 1"},
			map[string]string{"for delete 2": "test 2"},
			map[string]string{"Тестовый параметр": "Первое значение тестового параметра"},
			map[string]string{"Тестовый параметр": "Второе значение тестового параметра"},
			map[string]string{"Первый тестовый параметр": "Значение первого тестового параметра"},
			map[string]string{"Второй тестовый параметр": "Значение второго тестового параметра"},
		)
		user.DeleteParams("for delete 1", "for delete 2")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<a href='/user'>User page</a> | <a href='/user?registration'>Registration page</a> | <a href='/user?login'>Login page</a> | <a href='/user?logout'>Logout page</a><hr>")
		user := r.Context().Value(userContextName)
		if user == nil {
			fmt.Fprint(w, "User not login")
			return
		}
		fmt.Fprintf(w, "Hello, %s!", user.(baxtep.User).Name)
	})

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		ico, _ := base64.StdEncoding.DecodeString("AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAABILAAASCwAAAAAAAAAAAAByGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/8q2uP9yGSL/yra4/3IZIv/Ktrj/yra4/3IZIv9yGSL/yra4/8q2uP9yGSL/yra4/3IZIv/Ktrj/chki/3IZIv/Ktrj/chki/+je3/9yGSL/yra4/3IZIv/Ktrj/chki/8q2uP9yGSL/chki/8q2uP9yGSL/yra4/3IZIv9yGSL/yra4/+je3//Ktrj/chki/8q2uP9yGSL/yra4/3IZIv/Ktrj/yra4/3IZIv/Ktrj/yra4/3IZIv9yGSL/chki/+je3/9yGSL/yra4/3IZIv/Ktrj/chki/8q2uP9yGSL/yra4/3IZIv9yGSL/yra4/3IZIv/Ktrj/chki/3IZIv/Ktrj/chki/8q2uP9yGSL/yra4/8q2uP9yGSL/chki/8q2uP/Ktrj/chki/8q2uP/Ktrj/yra4/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/+je3//o3t//6N7f/+je3//o3t//6N7f/+je3/9yGSL/6N7f/+je3//o3t//6N7f/+je3//o3t//chki/3IZIv/o3t//yra4/8q2uP/Ktrj/yra4/8q2uP/Ktrj/chki/8q2uP/Ktrj/yra4/8q2uP/Ktrj/6N7f/3IZIv9yGSL/6N7f/8q2uP9yGSL/chki/3IZIv/Ktrj/6N7f/3IZIv/Ktrj/yra4/3IZIv9yGSL/yra4/+je3/9yGSL/chki/+je3//Ktrj/chki/8q2uP/o3t//6N7f/8q2uP9yGSL/6N7f/+je3/9yGSL/chki/8q2uP/o3t//chki/3IZIv/o3t//yra4/3IZIv/o3t//yra4/8q2uP/Ktrj/chki/8q2uP/Ktrj/chki/3IZIv/Ktrj/6N7f/3IZIv9yGSL/6N7f/+je3/9yGSL/chki/3IZIv9yGSL/chki/3IZIv/Ktrj/yra4/8q2uP/Ktrj/6N7f/+je3/9yGSL/chki/+je3//o3t//6N7f/+je3//o3t//6N7f/+je3/9yGSL/6N7f/+je3//o3t//6N7f/+je3//o3t//chki/3IZIv/Ktrj/yra4/8q2uP/Ktrj/yra4/8q2uP/Ktrj/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/chki/3IZIv9yGSL/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==")
		w.Write(ico)
	})

	return http.ListenAndServe(listenAddres, recoverLoggerHandler(baxtepHandler.Handler(mux)))
}

func recoverLoggerHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recover after panic %+v", r)
				http.Error(w, "Ooops!", http.StatusInternalServerError)
			}
		}()
		log.Printf("%s requested %s", r.RemoteAddr, r.URL)
		h.ServeHTTP(w, r)
	})
}

type LogWriter struct {}

func (l LogWriter) Write(p []byte) (n int, err error) {
	var w bytes.Buffer
	n, err = w.Write(p)
	fmt.Printf("BAXTER log: %s\n", w.String())
	return
}
package baxtep

var defaultTemplate = `
{{- define "_userregistration" -}}
{{- template "_userheader" -}}
  Registration page<hr/>
  {{if ._User}}
    Hello {{._User.Name}}! You already registred.
  {{else}}
    <form action="?registration" method="POST">
      <fieldset>
        <legend>Registration form</legend>
        <label for="name">Name:</label> 
        <input id="name" name="name" type="text" size="25" autofocus/><br/>
        <label for="email">Email:</label> 
          <input id="email" name="email" type="email" size="25" autofocus/><br/>
        <label for="password">Password:</label>    
          <input id="password" name="password" type="password" size="25" autocomplete="off"/><br/>
        <label for="retry-password">Retry password:</label>    
          <input id="retry-password" name="retry-password" type="password" size="25" autocomplete="off"/><br/>
        <input name="submit" type="submit" value="Submit" />
      </fieldset>
    </form>
  {{end}}
  <a href='?login'>Login</a><br/>
  <a href='?base'>User page</a>
{{- template "_userfooter" -}}
{{end}}

{{- define "_userconfirmation" -}}
{{- template "_userheader" -}}
  Confirmed and login
{{- template "_userfooter" -}}
{{end}}

{{- define "_userlogin" -}}
{{- template "_userheader" -}}
  Login page<hr/>
  {{if ._User}}
    <a href='?base'>User page</a><br/>
	<a href='?logout'>Logout</a><br/> 
  {{else}}
    <form action="?login" method="POST">
      <fieldset>
        <legend>Login form</legend>
        <label for="email">Email:</label> 
          <input id="email" name="email" type="email" size="25" autofocus/><br/>
        <label for="password">Password:</label>    
          <input id="password" name="password" type="password" size="25" autocomplete="off"/><br/>
        <input name="submit" type="submit" value="Submit" />
      </fieldset>
    </form>
  <a href='?registration'>Registration</a>
  {{end}}
{{- template "_userfooter" -}}
{{end}}


{{- define "_usercameout" -}}
{{- template "_userheader" -}}
  Came out page<hr/>
{{- template "_userfooter" -}}
{{end}}

{{- define "_userbase" -}}
{{- template "_userheader" -}}
  User page<hr/>
  {{if ._User}}
    <a href='?logout'>Logout</a><br/>
    Hello {{._User.Name}} you email {{._User.Email}} and enable {{._User.Enable}}<br/>
    Params:
    <hr/>
    <ul>
    {{range $key, $value :=._UserParams}}
      <li>
        "{{$key}}":
        <ol>
        {{range $value}}
          <li>{{.}}</li>
        {{else}}
          No value
        {{end}}
        </ol>
      </li>
    {{else}}
      No params
    {{end}}
    </ul>
  <hr/>
  {{else}}
    <a href='?login'>Login</a><br/>
    <a href='?registration'>Registration</a>
  {{end}}
{{- template "_userfooter" -}}
{{end}}


{{- define "_userheader" -}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{._Title}}</title>
</head>
<body>
{{- end -}}

{{- define "_userfooter" -}}
</body>
</html>
{{- end -}}
`

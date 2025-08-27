<meta charset="UTF-8">
{{range .messages -}}
<div class="message">
  <dl>
    <dt>From</dt>
    <dd>{{.Sender}}</dd>
    <dt>Date</dt>
    <dd>{{.Timestamp}}</dd>
  </dl>
  {{$msgtype := index .Content "msgtype" -}}
  {{if $msgtype -}}
    {{$msgtypeData := index $msgtype "data" -}}
    {{if eq $msgtypeData "m.text" -}}
      {{$body := index .Content "body" -}}
      {{if $body -}}
      <div class="body">{{index $body "data"}}</div>
      {{end -}}
    {{else if eq $msgtypeData "m.image" -}}
      {{$url := index .Content "url" -}}
      {{if $url -}}
      <div class="body"><img src="{{index $url "data"}}" /></div>
      {{end -}}
    {{else -}}
      <div class="error">Unknown message type: {{$msgtypeData}}</div>
    {{end -}}
  {{else -}}
    <div class="error">No message type found</div>
  {{end -}}
</div>
{{end}}

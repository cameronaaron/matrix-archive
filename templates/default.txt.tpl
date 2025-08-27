{{range .messages -}}
================================================================================
From: {{.Sender}}
Date: {{formatTime .Timestamp}}
{{$msgtype := index .Content "msgtype" -}}
{{if $msgtype -}}
Type: {{$msgtype}}

{{if eq $msgtype "m.text" -}}
{{$body := index .Content "body" -}}
{{if $body -}}
{{$body}}
{{end -}}
{{else if eq $msgtype "m.image" -}}
{{$body := index .Content "body" -}}
{{$url := index .Content "url" -}}
{{if $body -}}
Caption: {{$body}}
{{end -}}
{{if $url -}}
Image URL: {{$url}}
{{end -}}
{{else if eq $msgtype "m.video" -}}
{{$body := index .Content "body" -}}
{{$url := index .Content "url" -}}
{{if $body -}}
Caption: {{$body}}
{{end -}}
{{if $url -}}
Video URL: {{$url}}
{{end -}}
{{else if eq $msgtype "m.file" -}}
{{$body := index .Content "body" -}}
{{$url := index .Content "url" -}}
{{if $body -}}
Filename: {{$body}}
{{end -}}
{{if $url -}}
File URL: {{$url}}
{{end -}}
{{else if eq $msgtype "m.audio" -}}
{{$body := index .Content "body" -}}
{{$url := index .Content "url" -}}
{{if $body -}}
Caption: {{$body}}
{{end -}}
{{if $url -}}
Audio URL: {{$url}}
{{end -}}
{{else if eq $msgtype "m.notice" -}}
{{$body := index .Content "body" -}}
{{if $body -}}
Notice: {{$body}}
{{end -}}
{{else -}}
{{$body := index .Content "body" -}}
{{if $body -}}
{{$body}}
{{else -}}
[Unknown message type: {{$msgtype}}]
{{end -}}
{{end -}}
{{else -}}
{{$body := index .Content "body" -}}
{{if $body -}}
{{$body}}
{{else -}}
[No message content]
{{end -}}
{{end -}}

{{end}}

<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Matrix Archive</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .message {
            background: white;
            margin-bottom: 15px;
            padding: 15px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            border-left: 4px solid #007bff;
        }
        .message-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
            padding-bottom: 8px;
            border-bottom: 1px solid #eee;
        }
        .sender {
            font-weight: 600;
            color: #007bff;
        }
        .timestamp {
            color: #666;
            font-size: 0.9em;
        }
        .body {
            margin-top: 10px;
            word-wrap: break-word;
        }
        .body img {
            max-width: 100%;
            height: auto;
            border-radius: 4px;
            margin: 5px 0;
        }
        .body video {
            max-width: 100%;
            height: auto;
            border-radius: 4px;
            margin: 5px 0;
        }
        .file-link {
            display: inline-block;
            padding: 8px 12px;
            background: #28a745;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin: 5px 0;
        }
        .file-link:hover {
            background: #218838;
        }
        .message-type-indicator {
            background: #17a2b8;
            color: white;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.8em;
            margin-left: 10px;
        }
        .error {
            color: #dc3545;
            font-style: italic;
        }
        .event-id {
            font-size: 0.8em;
            color: #999;
            margin-top: 5px;
        }
        h1 {
            color: #333;
            text-align: center;
            margin-bottom: 30px;
        }
    </style>
</head>
<body>
    <h1>Matrix Room Archive</h1>
    
{{range .messages -}}
<div class="message">
    <div class="message-header">
        <span class="sender">{{.Sender}}</span>
        <span class="timestamp">{{formatTime .Timestamp}}</span>
    </div>
    
    {{$msgtype := index .Content "msgtype" -}}
    {{if $msgtype -}}
        <span class="message-type-indicator">{{$msgtype}}</span>
        {{if eq $msgtype "m.text" -}}
            {{$body := index .Content "body" -}}
            {{if $body -}}
                <div class="body">{{$body}}</div>
            {{end -}}
        {{else if eq $msgtype "m.image" -}}
            {{$body := index .Content "body" -}}
            {{$url := index .Content "url" -}}
            {{if $url -}}
                <div class="body">
                    {{if $body}}<p>{{$body}}</p>{{end}}
                    <img src="{{$url}}" alt="{{if $body}}{{$body}}{{else}}Image{{end}}" />
                </div>
            {{end -}}
        {{else if eq $msgtype "m.video" -}}
            {{$body := index .Content "body" -}}
            {{$url := index .Content "url" -}}
            {{if $url -}}
                <div class="body">
                    {{if $body}}<p>{{$body}}</p>{{end}}
                    <video controls>
                        <source src="{{$url}}" type="video/mp4">
                        Your browser does not support the video tag.
                    </video>
                </div>
            {{end -}}
        {{else if eq $msgtype "m.file" -}}
            {{$body := index .Content "body" -}}
            {{$url := index .Content "url" -}}
            {{if $url -}}
                <div class="body">
                    <a href="{{$url}}" class="file-link" download>
                        📁 {{if $body}}{{$body}}{{else}}Download File{{end}}
                    </a>
                </div>
            {{end -}}
        {{else if eq $msgtype "m.audio" -}}
            {{$body := index .Content "body" -}}
            {{$url := index .Content "url" -}}
            {{if $url -}}
                <div class="body">
                    {{if $body}}<p>{{$body}}</p>{{end}}
                    <audio controls>
                        <source src="{{$url}}" type="audio/mpeg">
                        Your browser does not support the audio element.
                    </audio>
                </div>
            {{end -}}
        {{else if eq $msgtype "m.notice" -}}
            {{$body := index .Content "body" -}}
            {{if $body -}}
                <div class="body" style="font-style: italic; color: #666;">{{$body}}</div>
            {{end -}}
        {{else -}}
            {{$body := index .Content "body" -}}
            {{if $body -}}
                <div class="body">{{$body}}</div>
            {{else -}}
                <div class="error">Unknown message type: {{$msgtype}}</div>
            {{end -}}
        {{end -}}
    {{else -}}
        {{$body := index .Content "body" -}}
        {{if $body -}}
            <div class="body">{{$body}}</div>
        {{else -}}
            <div class="error">No message content found</div>
        {{end -}}
    {{end -}}
</div>
{{end}}

</body>
</html>

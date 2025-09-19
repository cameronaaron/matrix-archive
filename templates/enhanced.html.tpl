<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Matrix Chat Archive - Enhanced</title>
    <style>
        * {
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.5;
            color: #1a202c;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            text-align: center;
            padding: 30px 0;
            color: white;
            margin-bottom: 30px;
        }

        .header h1 {
            font-size: 2.5rem;
            font-weight: 300;
            margin: 0 0 10px 0;
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
        }

        .header .subtitle {
            font-size: 1.1rem;
            opacity: 0.9;
        }

        .stats-bar {
            background: rgba(255, 255, 255, 0.15);
            border-radius: 8px;
            padding: 15px;
            margin: 20px 0;
            color: white;
            display: flex;
            justify-content: space-around;
            text-align: center;
        }

        .stat-item {
            flex: 1;
        }

        .stat-number {
            font-size: 1.5rem;
            font-weight: bold;
            display: block;
        }

        .chat-container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }

        .message {
            padding: 20px 24px;
            border-bottom: 1px solid #f1f5f9;
            transition: background-color 0.2s;
        }

        .message:hover {
            background-color: #f8fafc;
        }

        .message:last-child {
            border-bottom: none;
        }

        .message-header {
            display: flex;
            align-items: center;
            margin-bottom: 8px;
            gap: 12px;
        }

        .user-avatar {
            width: 40px;
            height: 40px;
            border-radius: 20px;
            background: linear-gradient(45deg, #667eea, #764ba2);
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: 600;
            font-size: 16px;
            flex-shrink: 0;
            border: 2px solid rgba(255, 255, 255, 0.2);
        }

        .user-info {
            flex: 1;
            min-width: 0;
        }

        .display-name {
            font-weight: 600;
            color: #2d3748;
            font-size: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .platform-badge {
            background: #4299e1;
            color: white;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 10px;
            font-weight: 500;
            text-transform: uppercase;
        }

        .platform-badge.discord {
            background: #5865f2;
        }

        .platform-badge.telegram {
            background: #0088cc;
        }

        .platform-badge.matrix {
            background: #0dbd8b;
        }

        .user-id {
            font-size: 12px;
            color: #718096;
            margin-top: 2px;
        }

        .timestamp {
            color: #a0aec0;
            font-size: 12px;
            white-space: nowrap;
        }

        .message-content {
            margin-left: 52px;
            margin-top: 8px;
        }

        .reactions-container {
            margin-top: 12px;
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
        }

        .reaction {
            background: #f7fafc;
            border: 1px solid #e2e8f0;
            border-radius: 16px;
            padding: 4px 8px;
            font-size: 12px;
            display: flex;
            align-items: center;
            gap: 4px;
            transition: all 0.2s;
        }

        .reaction:hover {
            background: #edf2f7;
            border-color: #cbd5e0;
        }

        .reaction-emoji {
            font-size: 14px;
        }

        .reaction-count {
            color: #4a5568;
            font-weight: 500;
        }

        .reply-indicator {
            background: #edf2f7;
            border-left: 3px solid #4299e1;
            padding: 8px 12px;
            margin-bottom: 12px;
            border-radius: 0 8px 8px 0;
            font-size: 12px;
            color: #4a5568;
        }

        .edit-indicator {
            color: #718096;
            font-size: 11px;
            font-style: italic;
            margin-top: 4px;
        }

        .message-body {
            background: #f8fafc;
            padding: 12px 16px;
            border-radius: 8px;
            margin: 8px 0;
            border: 1px solid #e2e8f0;
        }

        .message-type-badge {
            background: #48bb78;
            color: white;
            padding: 2px 6px;
            border-radius: 10px;
            font-size: 10px;
            font-weight: 500;
            text-transform: uppercase;
        }

        .message-type-m\.image {
            background: #ed8936;
        }

        .message-type-m\.video {
            background: #9f7aea;
        }

        .message-type-m\.file {
            background: #38b2ac;
        }

        .message-type-m\.audio {
            background: #f56565;
        }

        .formatted-content {
            margin-top: 8px;
        }

        .formatted-content p {
            margin: 8px 0;
        }

        .formatted-content code {
            background: #f7fafc;
            padding: 2px 4px;
            border-radius: 3px;
            font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
            font-size: 13px;
        }

        .formatted-content pre {
            background: #2d3748;
            color: #e2e8f0;
            padding: 12px;
            border-radius: 6px;
            overflow-x: auto;
            font-size: 13px;
            margin: 8px 0;
        }

        .meta-info {
            margin-top: 12px;
            color: #a0aec0;
            font-size: 11px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .event-id {
            font-family: monospace;
            background: #f1f5f9;
            padding: 2px 4px;
            border-radius: 3px;
            cursor: pointer;
        }

        .footer {
            text-align: center;
            color: white;
            opacity: 0.8;
            font-size: 14px;
            margin-top: 30px;
        }

        img {
            max-width: 100%;
            height: auto;
            border-radius: 8px;
            margin: 8px 0;
        }

        video {
            max-width: 100%;
            border-radius: 8px;
            margin: 8px 0;
        }

        audio {
            width: 100%;
            margin: 8px 0;
        }

        .file-attachment {
            display: inline-block;
            background: #4299e1;
            color: white;
            padding: 8px 12px;
            border-radius: 6px;
            text-decoration: none;
            font-size: 14px;
            margin: 8px 0;
        }

        .file-attachment:hover {
            background: #3182ce;
        }

        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }

            .header h1 {
                font-size: 2rem;
            }

            .message {
                padding: 12px 16px;
            }

            .message-content {
                margin-left: 0;
                margin-top: 8px;
            }

            .message-header {
                flex-wrap: wrap;
                gap: 8px;
            }

            .stats-bar {
                flex-direction: column;
                gap: 10px;
            }

            .stat-item {
                flex: none;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💬 Matrix Chat Archive - Enhanced</h1>
            <div class="subtitle">Comprehensive message history with real usernames</div>
            
            <div class="stats-bar">
                <div class="stat-item">
                    <span class="stat-number">{{len .}}</span>
                    <span>Messages</span>
                </div>
                <div class="stat-item">
                    <span class="stat-number">{{countUniqueUsers .}}</span>
                    <span>Users</span>
                </div>
                <div class="stat-item">
                    <span class="stat-number">{{countPlatforms .}}</span>
                    <span>Platforms</span>
                </div>
                <div class="stat-item">
                    <span class="stat-number">{{countReactions .}}</span>
                    <span>Reactions</span>
                </div>
            </div>
        </div>

        <div class="chat-container">
            {{range $index, $message := .}}
            <div class="message">
                <div class="message-header">
                    <div class="user-avatar">
                        {{if .UserAvatar}}{{.UserAvatar}}{{else}}{{if .DisplayName}}{{substr .DisplayName 0 1 | upper}}{{else}}?{{end}}{{end}}
                    </div>
                    <div class="user-info">
                        <div class="display-name">
                            {{.DisplayName}}
                            {{if .Platform}}
                                <span class="platform-badge {{.Platform | lower}}">{{.Platform}}</span>
                            {{end}}
                        </div>
                        <div class="user-id">{{.UserID}}</div>
                    </div>
                    <div class="timestamp">{{formatTime .Timestamp}}</div>
                    {{$msgtype := index .Content "msgtype"}}
                    {{if $msgtype}}
                        <span class="message-type-badge message-type-{{$msgtype}}">{{$msgtype}}</span>
                    {{end}}
                </div>

                <div class="message-content">
                    {{if .RepliesTo}}
                        <div class="reply-indicator">
                            ↳ Replying to {{.RepliesTo.DisplayName}}: {{.RepliesTo.Content | truncate 100}}
                        </div>
                    {{end}}
                    
                    {{$body := index .Content "body"}}
                    {{$formattedBody := index .Content "formatted_body"}}
                    {{$url := index .Content "url"}}
                    
                    {{if eq $msgtype "m.text"}}
                        <div class="message-body">
                            {{if $formattedBody}}
                                <div class="formatted-content">{{$formattedBody | safeHTML}}</div>
                            {{else}}
                                {{$body}}
                            {{end}}
                        </div>
                    {{else if eq $msgtype "m.image"}}
                        <div class="message-body">
                            {{if $body}}<p>{{$body}}</p>{{end}}
                            {{if $url}}
                                <img src="{{$url}}" alt="{{if $body}}{{$body}}{{else}}Image{{end}}" loading="lazy" />
                            {{end}}
                        </div>
                    {{else if eq $msgtype "m.video"}}
                        <div class="message-body">
                            {{if $body}}<p>{{$body}}</p>{{end}}
                            {{if $url}}
                                <video controls preload="metadata">
                                    <source src="{{$url}}" type="video/mp4">
                                    Your browser does not support the video tag.
                                </video>
                            {{end}}
                        </div>
                    {{else if eq $msgtype "m.file"}}
                        <div class="message-body">
                            {{if $url}}
                                <a href="{{$url}}" class="file-attachment" download>
                                    {{if $body}}{{$body}}{{else}}Download File{{end}}
                                </a>
                            {{else}}
                                {{$body}}
                            {{end}}
                        </div>
                    {{else if eq $msgtype "m.audio"}}
                        <div class="message-body">
                            {{if $body}}<p>{{$body}}</p>{{end}}
                            {{if $url}}
                                <audio controls preload="metadata">
                                    <source src="{{$url}}" type="audio/mpeg">
                                    Your browser does not support the audio element.
                                </audio>
                            {{end}}
                        </div>
                    {{else if eq $msgtype "m.notice"}}
                        <div class="message-body" style="font-style: italic; opacity: 0.8;">
                            {{if $formattedBody}}
                                <div class="formatted-content">{{$formattedBody | safeHTML}}</div>
                            {{else}}
                                {{$body}}
                            {{end}}
                        </div>
                    {{else}}
                        <div class="message-body">
                            {{if $body}}
                                {{$body}}
                            {{else}}
                                <em style="color: #a0aec0;">Unknown message type: {{$msgtype}}</em>
                            {{end}}
                        </div>
                    {{end}}

                    <div class="meta-info">
                        <span class="event-id" title="Event ID">{{.EventID}}</span>
                        <span>•</span>
                        <span title="Message Type">{{.MessageType}}</span>
                    </div>
                    
                    {{if .Reactions}}
                        <div class="reactions-container">
                            {{range .Reactions}}
                                <div class="reaction" title="{{range $i, $user := .Users}}{{if $i}}, {{end}}{{$user}}{{end}}">
                                    <span class="reaction-emoji">{{.Emoji}}</span>
                                    <span class="reaction-count">{{.Count}}</span>
                                </div>
                            {{end}}
                        </div>
                    {{end}}
                    
                    {{if .IsEdited}}
                        <div class="edit-indicator">
                            ✏️ Message edited
                        </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>

        <div class="footer">
            Generated by Matrix Archive with enhanced bridge mapping<br>
            <small>{{countBridgeUsers .}} Discord users mapped • {{len .}} total messages</small>
        </div>
    </div>
</body>
</html>
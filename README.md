# caddy-filemanager

#### What function does this module have?
add delete file options like this:
![view](/img/view.png)


#### Caddyfile config:
```Caddyfile
{
    order file_manager before file_server
}
:8080 {
    header {
        -Content-Security-Policy
        +Content-Security-Policy "default-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; connect-src 'self';"
        -Cache-Control
        +Cache-Control "no-cache, must-revalidate"
    }
    route {
        file_manager {
            root /your/path
        }
        file_server {
            root /your/path
            browse /path/to/this/repo/template.html
        }
    }
}
```

```shell
xcaddy build --with github.com/anwenzen/caddy-filemanager
```
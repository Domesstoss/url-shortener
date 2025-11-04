module url-shortener

go 1.24

require (
	github.com/fatih/color v1.15.0
	github.com/gavv/httpexpect/v2 v2.17.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/render v1.0.3
	github.com/go-playground/validator/v10 v10.28.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/brianvoe/gofakeit/v6 v6.28.0
	modernc.org/sqlite v1.39.1
)

replace github.com/hpcloud/tail => github.com/nxadm/tail v1.4.11

replace gopkg.in/fsnotify.v1 => github.com/fsnotify/fsnotify v1.7.0

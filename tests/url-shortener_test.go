package tests

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	mwlogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/lib/random"
	sqlite "url-shortener/internal/storage/sqlite"
)

const (
	host = "localhost:8082"
)

// helper — создаёт и запускает тестовый сервер
// поднимает сервер на localhost:8082 с БД в памяти (без файла)
func startTestServer(t *testing.T) func() {
	t.Helper()

	// in-memory DSN для modernc.org/sqlite
	// одна общая БД в памяти для процесса теста
	dsn := "file:memdb1?mode=memory&cache=shared&_busy_timeout=5000"

	st, err := sqlite.New(dsn)
	require.NoError(t, err)

	log := slogdiscard.NewDiscardLogger()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(mwlogger.New(log))
	r.Use(middleware.Recoverer)

	r.Post("/url", save.New(log, st))
	r.Get("/{alias}", redirect.New(log, st))

	srv := &http.Server{Addr: host, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("server failed: %v", err)
		}
	}()

	waitForPort(t, host)

	return func() {
		// аккуратно гасим HTTP и закрываем БД (на всякий)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = st.Close()
	}
}

func waitForPort(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 30; i++ {
		if c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server %s did not start in time", addr)
}

// --- твои тесты ---

func TestURLShortener_HappyPath(t *testing.T) {
	stop := startTestServer(t)
	defer stop()

	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(200).
		JSON().Object().
		ContainsKey("alias")
}

//nolint:funlen
func TestURLShortener_SaveRedirect(t *testing.T) {
	stop := startTestServer(t)
	defer stop()

	testCases := []struct {
		name  string
		url   string
		alias string
		error string
	}{
		{
			name:  "Valid URL",
			url:   gofakeit.URL(),
			alias: gofakeit.Word() + gofakeit.Word(),
		},
		{
			name:  "Invalid URL",
			url:   "invalid_url",
			alias: gofakeit.Word(),
			error: "field URL is not a valid URL",
		},
		{
			name:  "Empty Alias",
			url:   gofakeit.URL(),
			alias: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{Scheme: "http", Host: host}
			e := httpexpect.Default(t, u.String())

			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth("myuser", "mypass").
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")
				resp.Value("error").String().IsEqual(tc.error)
				return
			}

			alias := tc.alias
			if alias == "" {
				resp.Value("alias").String().NotEmpty()
				alias = resp.Value("alias").String().Raw()
			}

			testRedirect(t, alias, tc.url)
		})
	}
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)
	require.Equal(t, urlToRedirect, redirectedToURL)
}

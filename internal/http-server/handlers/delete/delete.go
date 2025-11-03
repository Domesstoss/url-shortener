package delete

import (
	"log/slog"
	"net/http"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type URLDeleter interface {
	DeleteURL(alias string) error
}

func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("empty alias in path")
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		if err := urlDeleter.DeleteURL(alias); err != nil {
			if err == storage.ErrURLNotFound {
				log.Error("failed to delete url", sl.Err(err))
				render.JSON(w, r, resp.Error("failed to delete url"))
				return
			}

			log.Error("failed to  delete url", sl.Err(err), slog.String("alias", alias))
			render.JSON(w, r, resp.Error("failed to delete url"))

			return

		}

		log.Info("url deleted", slog.String("alias", alias))
		render.JSON(w, r, resp.OK())
	}

}

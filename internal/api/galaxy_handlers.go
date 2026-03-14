package api

import (
	"net/http"
	"strconv"

	"galaxis/internal/db"
	"galaxis/internal/model"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerGalaxyRoutes(r chi.Router, pool *pgxpool.Pool) {
	r.Get("/galaxies", listGalaxies(pool))
	r.Get("/galaxy/{galaxyID}/stars", listStars(pool))
	r.Get("/galaxy/{galaxyID}/stars/{starID}", getStar(pool))
	r.Get("/galaxy/{galaxyID}/nebulae", listNebulae(pool))
}

func listGalaxies(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryGalaxies(r.Context(), pool)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if rows == nil {
			rows = []model.GalaxyRow{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"galaxies": rows})
	}
}

// listStars returns paginated stars for a galaxy within an optional bounding box.
// Query params: x1,y1,z1,x2,y2,z2 (ly), limit (default 5000, max 10000), offset.
func listStars(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		galaxyID, err := uuid.Parse(chi.URLParam(r, "galaxyID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid galaxy id")
			return
		}

		q := r.URL.Query()
		x1 := parseFloatOr(q.Get("x1"), -1e9)
		y1 := parseFloatOr(q.Get("y1"), -1e9)
		z1 := parseFloatOr(q.Get("z1"), -1e9)
		x2 := parseFloatOr(q.Get("x2"), 1e9)
		y2 := parseFloatOr(q.Get("y2"), 1e9)
		z2 := parseFloatOr(q.Get("z2"), 1e9)
		limit := parseIntOr(q.Get("limit"), 5000)
		if limit > 10000 {
			limit = 10000
		}
		offset := parseIntOr(q.Get("offset"), 0)

		stars, err := db.QueryStarsBbox(r.Context(), pool, galaxyID,
			x1, y1, z1, x2, y2, z2, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if stars == nil {
			stars = []model.StarRow{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"stars":  stars,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func getStar(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		starID, err := uuid.Parse(chi.URLParam(r, "starID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid star id")
			return
		}
		star, err := db.QueryStarByID(r.Context(), pool, starID)
		if err != nil {
			writeError(w, http.StatusNotFound, "star not found")
			return
		}
		writeJSON(w, http.StatusOK, star)
	}
}

func listNebulae(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		galaxyID, err := uuid.Parse(chi.URLParam(r, "galaxyID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid galaxy id")
			return
		}
		nebulae, err := db.QueryNebulae(r.Context(), pool, galaxyID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if nebulae == nil {
			nebulae = []model.NebulaRow{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"nebulae": nebulae})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseFloatOr(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

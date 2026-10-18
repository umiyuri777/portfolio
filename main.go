package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// embeddedTemplates は templates/ 配下の HTML をバイナリへ埋め込むことで、
// ランタイムでファイル配置を意識しなくて済むようにする
// （Dockerfileの COPY ルールを増やさないために embed を選択している）。
//
//go:embed templates/*.html
var embeddedTemplates embed.FS

// templateRenderer is a small adapter that satisfies echo.Renderer.
type templateRenderer struct {
	templates *template.Template
}

func (t *templateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// newRenderer parses every embedded template once at startup.
// Echoの標準レンダラはfuncMapを後付けしにくいので、明示的に組み立てている。
func newRenderer() (*templateRenderer, error) {
	funcs := template.FuncMap{
		// skillBar renders a 5-stage proficiency bar using block characters.
		// Notion側のスキル指標と表示を一致させるため、外で定数化した上限を使う。
		"skillBar": func(level int) template.HTML {
			if level < 0 {
				level = 0
			}
			if level > MaxSkillLevel {
				level = MaxSkillLevel
			}
			filled := strings.Repeat("█", level)
			empty := strings.Repeat("░", MaxSkillLevel-level)
			return template.HTML(filled + `<span class="skill__bar-empty">` + empty + `</span>`)
		},
		// rank turns a 0-based range index into a zero-padded ranking ("01", "02", ...).
		"rank": func(i int) string {
			return fmt.Sprintf("%02d", i+1)
		},
		// jst formats timestamps in JST. alpine には tzdata が無いので固定オフセットを使う。
		"jst": func(t time.Time) string {
			return t.In(time.FixedZone("JST", 9*60*60)).Format("2006-01-02 15:04 JST")
		},
	}

	tmpl := template.New("").Funcs(funcs)
	parsed, err := tmpl.ParseFS(embeddedTemplates, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &templateRenderer{templates: parsed}, nil
}

// pageData is what index.html renders: the static resume plus live Spotify stats.
// Resume を埋め込むことで、テンプレート側は従来どおり {{ .Handle }} で参照できる。
type pageData struct {
	Resume
	Spotify *SpotifyStats
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func main() {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	renderer, err := newRenderer()
	if err != nil {
		log.Fatalf("failed to initialize templates: %v", err)
	}
	e.Renderer = renderer

	// 静的アセットは /static で配信。Dockerfileの既存COPYルールで対応済み。
	e.Static("/static", "public")

	resume := BuildResume()
	// 認証情報が未設定なら nil になり、Spotify セクションは表示されない。
	spotify := NewSpotifyClientFromEnv()

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", pageData{
			Resume:  resume,
			Spotify: spotify.Stats(),
		})
	})

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	port := getEnv("PORT", "8080")
	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

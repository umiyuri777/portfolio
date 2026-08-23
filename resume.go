package main

// Resume holds all CV-style content rendered on the portfolio page.
//
// Notionの「Resume」ページを単一のソース・オブ・トゥルースとして転記している。
// 値の更新が必要になった場合は、ここを書き換えるだけで全画面に反映される。
type Resume struct {
	Handle    string
	Name      string
	Headline  string
	UpdatedAt string

	Summary    []string
	Education  []TimelineItem
	Experience []TimelineItem

	Projects []Project
	Skills   []SkillGroup
	Links    []Link
}

// TimelineItem represents a single row in the education/experience sections.
type TimelineItem struct {
	When  string
	Title string
	Notes []string
}

// Project represents one entry of the portfolio collection.
type Project struct {
	Name        string
	Description string
	Repo        string
}

// SkillGroup groups skills under a common category for readable rendering.
type SkillGroup struct {
	Category string
	Items    []Skill
}

// Skill is a single skill entry with a 5-stage proficiency level.
type Skill struct {
	Name   string
	Level  int
	Detail string
}

// Link is an external link displayed in the contact section.
type Link struct {
	Label string
	URL   string
}

// MaxSkillLevel defines the upper bound used when rendering skill bars.
// Notion側の「skill（5段階）」と一致させるためマジックナンバー化を避けて定数化している。
const MaxSkillLevel = 5

// BuildResume returns the static resume payload used by the template renderer.
//
// Notion ID: 231cc6e1-e247-8008-a9a5-f311ddef1847
// 値の出典: Notion公開ページ "Resume" (2026年5月時点のスナップショット)
func BuildResume() Resume {
	return Resume{
		Handle:    "umiyuri777",
		Name:      "umiyuri",
		Headline:  "立命館大学大学院情報理工学研究科 1年生 / バックエンド志向のソフトウェアエンジニア",
		UpdatedAt: "2026-05",

		Summary: []string{
			"立命館大学大学院情報理工学研究科の1年生．個人での開発経験とサークル活動を通じて技術を磨いてきた．",
			"強みは、泥臭く頑張る粘り強さとコミュニケーション力．",
			"開発でうまくいかないことがあっても、誰よりも問題解決に時間を使い、誰よりも多く試行錯誤して粘り強く取り組んできた．",
			"競技プログラミング（AtCoder）でも、解けない問題はしっかり理解するまで考え抜く姿勢を続け、茶色レートに到達．",
			"チーム開発では認識の齟齬を防ぐため、曖昧な部分を必ず議題に出し、抽象的な点を残さず具体化することを徹底している．",
		},

		Education: []TimelineItem{
			{When: "2022.03", Title: "京都成章高校 卒業"},
			{When: "2022.04", Title: "立命館大学 情報理工学部 入学"},
			{When: "2026.03", Title: "立命館大学 情報理工学部 卒業"},
			{When: "2026.04", Title: "立命館大学院 情報理工学研究科 入学"},
		},

		Experience: []TimelineItem{
			{
				When:  "2023 ~ 2026",
				Title: "プログラミングサークル「watnow」 所属",
				Notes: []string{
					"エンジニアリング能力を高めるため、同じ目標を持つ仲間を求めて所属",
				},
			},
			{
				When:  "2025 ~",
				Title: "ライフイズテック株式会社 インターン",
				Notes: []string{
					"プログラミングだけでなく、コミュニケーション能力・チームビルディング能力を実務で習得",
				},
			},
			{
				When:  "2025.11",
				Title: "技育展2025 決勝大会 出場",
				Notes: []string{
					"3ヶ月にわたる5人でのチーム開発に取り組む",
					"プレゼンにて120人規模でのデモを実施",
				},
			},
			{
				When:  "2026.02",
				Title: "ピクシブ株式会社 就業型インターン（2/17 - 2/27）",
				Notes: []string{
					"ピクシブ百科事典チームで新機能のバックエンド開発に参加",
					"技術的負債の解消・ライブラリ選定など意思決定から実装まで担当",
					"一つ一つの意思決定に納得できる理由を用意する重要性を学ぶ",
				},
			},
			{
				When:  "2026.04 ~",
				Title: "株式会社Finatext 就業型長期インターン",
				Notes: []string{
					"証券チームのバックエンド開発に参画",
				},
			},
		},

		Projects: []Project{
			{
				Name:        "むきむきくらげ",
				Description: "筋トレ継続支援アプリ。ジムに行ったことを記録するとクラゲの成長ゲージが溜まり、ユーザー全員でクラゲを育てる。",
				Repo:        "https://github.com/Muscle-Misalignment/mukimuki-kurage",
			},
			{
				Name:        "AtCoder今日の1問bot",
				Description: "競技プログラミング向けDiscord Bot。毎朝9時にAtCoderの過去問から1問をランダム抽出してURLを投稿する。",
				Repo:        "https://github.com/umiyuri777/ACprobremsRemainder",
			},
			{
				Name:        "Streamerio",
				Description: "ゲーム配信者と視聴者が密に交流する新感覚ライブゲーム。視聴者が配信者のゲームに直接関与し、配信者を「守る」のか「倒す」のかでUXを変える。",
				Repo:        "https://github.com/Streamerio/Streamerio",
			},
			{
				Name:        "DJBN（デジャブんじゃねえ）",
				Description: "似ているサービス案を自動検索してデジャブを防ぐ。",
				Repo:        "https://github.com/umiyuri777/DJBN-server",
			},
			{
				Name:        "観客参加型VJsystem",
				Description: "観客が参加できる新感覚参加型VJシステム。",
				Repo:        "https://github.com/umiyuri777/VJsystem-front",
			},
		},

		Skills: []SkillGroup{
			{
				Category: "Programming Languages & Tools",
				Items: []Skill{
					{Name: "Go", Level: 3, Detail: "サークルやハッカソンでのバックエンド開発経験"},
					{Name: "Python", Level: 3, Detail: "競技プログラミング、Flask/FastAPIでのAPI作成"},
					{Name: "GCP", Level: 3, Detail: "Google Cloud Platformを用いたサーバー構築経験"},
					{Name: "AWS", Level: 3, Detail: "インターンでのAWSを用いたサーバー構築経験"},
					{Name: "Unity (C#)", Level: 3, Detail: "ゲーム制作経験。アルバイトで中高生への指導経験あり"},
					{Name: "Flutter (Dart)", Level: 2, Detail: "サークルでのアプリ制作経験"},
					{Name: "C", Level: 2, Detail: "大学の授業"},
				},
			},
			{
				Category: "Languages",
				Items: []Skill{
					{Name: "日本語", Level: 5, Detail: "母国語"},
					{Name: "English", Level: 1, Detail: ""},
				},
			},
		},

		Links: []Link{
			{Label: "GitHub", URL: "https://github.com/umiyuri777"},
			{Label: "AtCoder", URL: "https://atcoder.jp/users/umiyuri"},
			{Label: "Qiita", URL: "https://qiita.com/umiyuri777"},
		},
	}
}

package render

// Site — данные, общие для всех страниц: пилюля со звёздами и строка в футере.
type Site struct {
	Repo      string // "Sysoev973/Proks"
	Stars     int
	SHA       string // коммит Proks, из которого собран сайт
	CommitURL string
	Year      int
}

// ShortSHA — первые 7 символов коммита, как их показывает GitHub.
func (s Site) ShortSHA() string {
	if len(s.SHA) > 7 {
		return s.SHA[:7]
	}
	return s.SHA
}

// HomeData — данные шаблона index.html.
type HomeData struct {
	GoVersion string
	Modules   []Module
	Chips     []Chip
}

// Module — карточка пакета internal/<Name>. Описание берётся из словаря по ключу mod.<Name>.
type Module struct {
	Name  string
	Badge string // "CORE", "NEW" или пусто
	Tags  []string
	Tests int // файлов *_test.go в пакете
	URL   string
}

// Chip — быстрый переход в раздел документации. Label — ключ словаря.
type Chip struct {
	Label string
	URL   string
}

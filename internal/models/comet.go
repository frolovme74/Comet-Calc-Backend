package models

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusDeleted   = "deleted"
)

type Comet struct {
	ID               int
	CometName        string
	CometDescription string

	OrbitalPeriod     float64
	OrbitEccentricity float64

	CometPhoto string
	CometVideo string

	CometStatus string
	Likes       []int
}

var Comets = []Comet{
	{
		ID:        1,
		CometName: "1P/Галлея",
		CometDescription: "Самая известная короткопериодическая комета: возвращается к Солнцу " +
			"примерно раз в 76 лет. Последний перигелий — 9 февраля 1986 года, тогда её " +
			"вблизи впервые сфотографировал аппарат «Джотто». Следующее возвращение — 2061 год.",
		OrbitalPeriod:     75.92,
		OrbitEccentricity: 0.968,
		CometPhoto:        "halley.jpg",
		CometVideo:        "halley.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{1, 2, 3, 5, 8, 13},
	},
	{
		ID:        2,
		CometName: "2P/Энке",
		CometDescription: "Комета с самым коротким периодом среди ярких комет — около 3,3 года. " +
			"Впервые замечена Пьером Мешеном в 1786 году, орбиту рассчитал Иоганн Энке. " +
			"С её пылевым шлейфом связан метеорный поток Тауриды.",
		OrbitalPeriod:     3.31,
		OrbitEccentricity: 0.847,
		CometPhoto:        "encke.jpg",
		CometVideo:        "encke.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{2, 8},
	},
	{
		ID:        3,
		CometName: "67P/Чурюмова — Герасименко",
		CometDescription: "Комета, открытая советскими астрономами в 1969 году. Цель миссии ESA " +
			"«Розетта»: в 2014 году на её ядро впервые в истории сел спускаемый аппарат «Филы».",
		OrbitalPeriod:     6.44,
		OrbitEccentricity: 0.641,
		CometPhoto:        "churyumov.jpg",
		CometVideo:        "churyumov.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{2, 3, 6},
	},
	{
		ID:        4,
		CometName: "12P/Понса — Брукса",
		CometDescription: "Комета с периодом около 71 года, известная вспышками яркости: " +
			"из-за формы комы в 2023 году её прозвали «рогатой». Прошла перигелий 21 апреля 2024 года.",
		OrbitalPeriod:     71.24,
		OrbitEccentricity: 0.955,
		CometPhoto:        "pons_brooks.jpg",
		CometVideo:        "pons_brooks.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{5, 6},
	},
	{
		ID:        5,
		CometName: "109P/Свифта — Туттля",
		CometDescription: "Родительская комета метеорного потока Персеиды. Ядро диаметром около 26 км — " +
			"крупнейший объект, регулярно сближающийся с Землёй. Последний перигелий — 1992 год, " +
			"следующий — 2126 год.",
		OrbitalPeriod:     133.28,
		OrbitEccentricity: 0.963,
		CometPhoto:        "swift_tuttle.jpg",
		CometVideo:        "swift_tuttle.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{1, 3, 5, 7, 9, 11, 12},
	},
	{
		ID:        6,
		CometName: "9P/Темпеля",
		CometDescription: "В 2005 году аппарат NASA «Дип Импакт» сбросил на ядро кометы медный " +
			"ударник массой около 370 кг, чтобы изучить вещество под поверхностью. " +
			"В 2011 году комету повторно посетил «Стардаст».",
		OrbitalPeriod:     5.58,
		OrbitEccentricity: 0.51,
		CometPhoto:        "tempel1.jpg",
		CometVideo:        "tempel1.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{1, 4, 7, 9, 10},
	},
	{
		ID:        7,
		CometName: "55P/Темпеля — Туттля",
		CometDescription: "Родительская комета метеорного потока Леониды. Открыта независимо " +
			"Эрнстом Темпелем и Хорасом Туттлем в 1865–1866 годах. Последний перигелий — " +
			"1998 год, следующий — 2031 год.",
		OrbitalPeriod:     33.24,
		OrbitEccentricity: 0.906,
		CometPhoto:        "tempel_tuttle.jpg",
		CometVideo:        "tempel_tuttle.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{1, 2, 3, 4},
	},
	{
		ID:        8,
		CometName: "103P/Хартли",
		CometDescription: "Небольшая комета с ядром в форме арахиса длиной около 2 км. " +
			"В ноябре 2010 года её снял с расстояния около 700 км аппарат NASA EPOXI " +
			"(бывший «Дип Импакт»).",
		OrbitalPeriod:     6.48,
		OrbitEccentricity: 0.694,
		CometPhoto:        "hartley2.jpg",
		CometVideo:        "hartley2.mp4",
		CometStatus:       StatusPublished,
		Likes:             []int{4},
	},
	{
		ID:        9,
		CometName: "81P/Вильда",
		CometDescription: "В 2004 году аппарат NASA «Стардаст» пролетел сквозь кому кометы и собрал " +
			"частицы пыли, которые в 2006 году доставил на Землю.",
		OrbitalPeriod:     6.41,
		OrbitEccentricity: 0.537,
		CometPhoto:        "wild2.jpg",
		CometVideo:        "wild2.mp4",
		CometStatus:       StatusDraft,
		Likes:             []int{},
	},
	{
		ID:        10,
		CometName: "46P/Виртанена",
		CometDescription: "В декабре 2018 года прошла в 0,08 а.е. от Земли — одно из самых " +
			"тесных сближений кометы с Землёй за последние десятилетия.",
		OrbitalPeriod:     5.44,
		OrbitEccentricity: 0.659,
		CometPhoto:        "wirtanen.jpg",
		CometVideo:        "wirtanen.mp4",
		CometStatus:       StatusDeleted,
		Likes:             []int{1},
	},
}

func PublishedComets() []Comet {
	res := make([]Comet, 0, len(Comets))
	for _, c := range Comets {
		if c.CometStatus == StatusPublished {
			res = append(res, c)
		}
	}
	return res
}

func PublishedCometByID(id int) (Comet, bool) {
	for _, c := range PublishedComets() {
		if c.ID == id {
			return c, true
		}
	}
	return Comet{}, false
}

func NextPublishedComet(id int) (Comet, bool) {
	list := PublishedComets()
	for i, c := range list {
		if c.ID == id {
			return list[(i+1)%len(list)], true
		}
	}
	return Comet{}, false
}

func DraftComet() (Comet, bool) {
	for _, c := range Comets {
		if c.CometStatus == StatusDraft {
			return c, true
		}
	}
	return Comet{}, false
}

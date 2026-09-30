# Comet-Calc-Backend — расстояние кометы от Солнца (ЛР2)

Тема: расстояние кометы от Солнца по её координатам на небесной сфере.
Услуга — **комета**, заявка — **расчёт** расстояния (появится в следующих ЛР).

Поля услуги по теме: период обращения P (лет) и эксцентриситет орбиты e. Фильтрация — по периоду (от и до).

ЛР2: данные хранятся в PostgreSQL, три страницы из ЛР1 работают с БД. Получение, поиск, создание и публикация
услуг — через ORM (gORM), логическое удаление — SQL-запросом `UPDATE` без ORM. Текущий пользователь
зафиксирован константой `CurrentUserID = 1` (авторизация — в ЛР4). Лайки только отображаются.

Бэкенд: Go 1.25, `net/http` + `html/template`, gORM + PostgreSQL, godotenv; Minio для фото и видео.

## Структура

```
cmd/comets/main.go                 сервер: роутинг 6 HTTP-методов, статика, функции шаблонов
cmd/migrate/main.go                миграции: создание таблиц по моделям (gORM AutoMigrate)
internal/models/                   модели = таблицы: User, Comet, CometLike
internal/repository/               работа с БД: запросы через gORM, удаление через SQL UPDATE
internal/handlers/handlers.go      контроллеры
internal/media/media.go            фото и видео по умолчанию, проверка доступности файлов в Minio
internal/dsn/dsn.go                строка подключения к PostgreSQL из переменных окружения
templates/                         comet_feed.html, comet_draft.html, comet_list.html, error.html, partials.html
static/css/style.css               стили (скопированы с TheSkyLive)
static/media/                      фото и видео по умолчанию
db/seed.sql                        начальные данные для Adminer
docker-compose.yml                 Minio, PostgreSQL, Adminer
.env.example                       параметры подключения к БД (скопировать в .env)
```

## База данных

Каскадное удаление не используется: все внешние ключи `ON DELETE RESTRICT`.

**users** — пользователи

| Столбец | Тип | |
|---|---|---|
| id | bigint | первичный ключ |
| login | varchar(50) | уникальный |
| full_name | varchar(100) | |

**comets** — услуги (кометы)

| Столбец | Тип | |
|---|---|---|
| id | bigint | первичный ключ |
| comet_name | varchar(100) | наименование |
| comet_description | varchar(500) | краткое описание |
| comet_status | varchar(20) | `draft` / `published` / `deleted` (CHECK) |
| comet_photo | varchar(255) | url изображения в Minio |
| comet_video | varchar(255) | url видео в Minio |
| orbital_period | numeric(8,2) | период обращения, лет (> 0) |
| orbit_eccentricity | numeric(4,3) | эксцентриситет (0 ≤ e < 1) |
| created_at | timestamp | дата создания |
| creator_id | bigint | создатель → users.id |
| formed_at | timestamp | дата формирования (публикации) |

Уникальный частичный индекс `idx_comets_one_draft_per_creator (creator_id) WHERE comet_status = 'draft'` —
у каждого пользователя не более одного черновика.

**comet_likes** — лайки, м-м пользователь–комета

| Столбец | Тип | |
|---|---|---|
| user_id | bigint | первичный ключ (составной) → users.id |
| comet_id | bigint | первичный ключ (составной) → comets.id |

## Запуск

1. Параметры БД: `cp .env.example .env` (PostgreSQL на порту 5433, чтобы не конфликтовать с другими базами).
2. Файлы фото и видео — в `media/` (список и источники — в разделе «Медиа: источники»).
3. Контейнеры: `docker compose up -d` — Minio (http://localhost:9001, `minioadmin`), PostgreSQL (localhost:5433),
   Adminer (http://localhost:8082: система PostgreSQL, сервер `postgres`, пользователь/пароль/база из `.env`).
   Контейнер `minio-init` создаёт бакет `comets`, открывает его на чтение и загружает файлы из `media/`.
4. Таблицы: `go run ./cmd/migrate`.
5. Данные: в Adminer → «SQL-запрос» → содержимое `db/seed.sql` → «Выполнить»
   (8 пользователей, 10 комет: 8 опубликованных, 1 черновик, 1 удалённая, 31 лайк).
6. Сервер: `go run ./cmd/comets` → http://localhost:8080/comets
   С выводом SQL-запросов в консоль (видно `LIMIT 1` ленты и `WHERE` фильтра): `DB_LOG=1 go run ./cmd/comets`.

Образы `minio/minio` и `minio/mc` удалены с Docker Hub, поэтому используется форк `pgsty/minio` + `pgsty/mc`.

## HTTP-методы

| Метод и URL | Контроллер | Работа с БД | Что делает |
|---|---|---|---|
| `GET /comets/feed/`, `/comets/feed/{id}`, `?next=true` | `CometFeed` | gORM, `LIMIT 1` | лента: первая / по id / следующая опубликованная комета |
| `GET /comets/draft` | `CometDraft` | gORM | «Добавление»: нет черновика — форма создания, есть — форма публикации |
| `GET /comets?min_orbital_period=&max_orbital_period=` | `CometList` | gORM, `WHERE` в SQL | плитка опубликованных с фильтром по периоду |
| `POST /comets/draft` | `CreateCometDraft` | gORM `Create` | кнопка «Далее»: создать черновик по названию |
| `POST /comets/draft/publish` | `PublishCometDraft` | gORM `Updates` | кнопка «Опубликовать»: описание, период, эксцентриситет, статус, дата формирования |
| `POST /comets/{id}/delete` | `DeleteComet` | SQL `UPDATE` без ORM | кнопка удаления на плитке: статус `deleted` |

- Лента: БД возвращает ровно одну строку (`LIMIT 1`); следующая — `id > ? ORDER BY id LIMIT 1`, после последней — первая.
- Фильтр: `WHERE orbital_period >= ? AND orbital_period <= ?` выполняется в БД. Слайдеры «от» и «до» — `input type="range"`,
  число над ползунком обновляется без JavaScript (CSS scroll-driven animations). Крайние значения 0 и 140 — без границы.
- Лайки: `(SELECT COUNT(*) FROM comet_likes WHERE comet_id = comets.id)` в том же запросе.
- Удаление: `UPDATE comets SET comet_status = 'deleted' WHERE id = $1 AND comet_status = 'published'` через `database/sql`.
  Удалённые кометы нигде не показываются: `/comets/feed/{id}` удалённой → 404.
- Создание: файлы фото и видео в ЛР2 на сервер не передаются (у полей выбора файла нет `name`), в БД url пустые.
- Фото и видео по умолчанию (`static/media/default_comet.jpg`, `default_comet.mp4`) подставляются, если url в БД пустой
  или файл в Minio недоступен (HEAD-запрос, результат кешируется на 30 с). У видео вторым `<source>` всегда указан ролик по умолчанию.
- Проверка ввода: название до 100 символов; при публикации — описание до 500 символов, период > 0 (до 999 999,99),
  эксцентриситет от 0 до 0,999 (значения округляются до точности столбцов). Ошибки показываются на странице по-русски.
- Ошибки 404 и 405 (несуществующий адрес, удалённая комета, неверный метод) — страница на русском `error.html`.

## Дизайн — источник

https://theskylive.com/comets — фон `#212121`, текст `#BBBBBB`, карточки `#3A3A3A` (скругление 8px),
кнопка `#007BFF` с белым текстом (кнопка языка в хедере), поле ввода `#404040` (поиск в хедере),
панель вкладок — как навигация в хедере: чёрный фон, ссылки `#F5BB4A` капителью,
активный пункт `#11283C` с линией `#4182BB`; заголовок `#89DDD7`.
Hover: карточка — фон `#7A4400` + рамка `#F5BB4A`, изображение — рамка `#444444`, синяя кнопка и пункт
навигации — `#3B3E8B`, светлая кнопка — `#E0E0E0`/`#222222`, ссылка — `#FFFF00`, фокус поля — `#AAAAAA`.
Полный список — в комментарии в начале `static/css/style.css`.

## Данные комет

Период и эксцентриситет — оскулирующие элементы орбит JPL Horizons (эпоха у каждой кометы своя).

## Медиа: источники

Фото — из открытых архивов NASA. Видео — отрезки настоящих съёмок и роликов NASA/ESA, переложенные
через ffmpeg в вертикаль 1080×1920 (видео целиком по центру, фон — оно же, размытое), без звука.

**Фото**

| Файл | Что на снимке | Источник | Подпись (credit) |
|---|---|---|---|
| halley.jpg | ядро Галлеи, «Джотто», 1986 | images.nasa.gov, PIA17485 | NASA/ESA/Giotto Project |
| encke.jpg | Энке и её пылевой шлейф, «Спитцер» | images.nasa.gov, PIA07222 | NASA/JPL-Caltech/Univ. of Minn. |
| churyumov.jpg | поверхность 67P, «Розетта», 29.09.2016 | images.nasa.gov, PIA21068 | ESA/Rosetta/MPS for OSIRIS Team (CC BY-SA 3.0 IGO) |
| swift_tuttle.jpg | метеор Персеид (поток кометы), 2021 | images.nasa.gov, NHQ202108110003 | NASA/Bill Ingalls |
| tempel1.jpg | вспышка после удара «Дип Импакт», 2005 | images.nasa.gov, PIA02137 | NASA/JPL-Caltech/UMD |
| hartley2.jpg | ядро Хартли 2, EPOXI, 2010 | images.nasa.gov, PIA13570 | NASA/JPL-Caltech/UMD |
| wild2.jpg | ядро Вильда 2, «Стардаст», 2004 | images.nasa.gov, PIA06285 | NASA/JPL-Caltech |
| wirtanen.jpg | Виртанен, «Хаббл», 13.12.2018 | science.nasa.gov | NASA, ESA, D. Bodewits (Auburn University), J.-Y. Li (PSI) |
| pons_brooks.jpg | комета 12P с хвостом, 2024 (кадр из ролика) | NASA JPL, «What's Up», март 2024 (1:59) | © Dan Bartlett, показано NASA/JPL |
| tempel_tuttle.jpg | болид Леонид над лесом (кадр из ролика) | NASA JPL, «What's Up», ноябрь 2023 (1:53) | © Ed Sweeney, показано NASA/JPL |

**Видео**

| Файл | Что в видео | Источник (отрезок) | Подпись (credit) |
|---|---|---|---|
| halley.mp4 | подлёт «Джотто» к ядру, 1986 | ESA, «Giotto 25 years movie» (3:04–3:24) | © MPS 1986–2011 (H. U. Keller et al.) |
| encke.mp4 | выброс Солнца срывает хвост кометы, STEREO, 2007 | NASA SVS, 20127 (целиком) | NASA/Goddard Scientific Visualization Studio |
| churyumov.mp4 | ядро и поверхность 67P, «Розетта» | ESA, «Rosetta's ever-changing view of a comet» (0:04–0:24) | ESA/Rosetta/NavCam — CC BY-SA IGO 3.0; ESA/Rosetta/MPS for OSIRIS Team |
| pons_brooks.mp4 | снимок кометы 12P с хвостом (замедлено) | NASA JPL, «What's Up», март 2024 (1:59–2:06) | © Dan Bartlett (снимок), NASA/JPL-Caltech |
| swift_tuttle.mp4 | метеоры Персеид, камера ESA на Канарах, 2020 | ESA, «Peak of Perseid showers…» (0:07–0:27) | ESA/Meteor Research Group/CILBO |
| tempel1.mp4 | пролёт «Стардаста» у ядра, 2011 | NASA Photojournal, PIA13867 (0:03–0:12) | NASA/JPL-Caltech/Cornell |
| tempel_tuttle.mp4 | ночное небо в поток Леонид | NASA JPL, «What's Up», ноябрь 2023 (1:52–2:01) | любительские снимки (авторы указаны в ролике), NASA/JPL-Caltech |
| hartley2.mp4 | пролёт EPOXI у ядра, 2010 (замедлено) | NASA Photojournal, PIA13602 (целиком) | NASA/JPL-Caltech/UMD |
| wild2.mp4 | анимация: «Стардаст» в коме кометы | NASA KSC, сюжет 2004 (0:14–0:30) | NASA |
| wirtanen.mp4 | снимок кометы 46P и её путь в декабре 2018 | NASA JPL, «What's Up», декабрь 2018 (1:10–1:26) | NASA/JPL-Caltech |

Материалы NASA свободны для использования с указанием источника. Материалы ESA с пометкой CC BY-SA
требуют подписи. Под копирайтом авторов (используются в учебном проекте, подпись обязательна):
видео Галлеи (MPS), фото и видео Понса — Брукса и Темпеля — Туттля (любительские снимки из роликов NASA).

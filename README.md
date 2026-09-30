# Comet-Calc-Backend — расстояние кометы от Солнца (ЛР3)

Тема: расстояние кометы от Солнца по её координатам на небесной сфере.
Услуга — **комета**, заявка — **расчёт** расстояния.

Поля услуги по теме: период обращения P (лет) и эксцентриситет орбиты e. Фильтрация — по периоду (от и до).

ЛР3: веб-сервис (REST API, JSON) со всей бизнес-логикой, кроме авторизации. Все методы начинаются с `/api`.
Работа с БД — через ORM (gORM), фото и видео загружаются файлами в Minio, в БД сохраняются имена файлов.
Текущий пользователь зафиксирован константой через функцию-singleton `session.CurrentUser()` (id = 1).
Страницы из ЛР1–ЛР2 (`/comets`, `/comets/feed`, `/comets/draft`) продолжают работать.

Бэкенд: Go 1.25, `net/http`, gORM + PostgreSQL, minio-go, bcrypt, godotenv.

## Структура

```
cmd/comets/main.go                   сервер: страницы (SSR) и API, статика
cmd/migrate/main.go                  миграции: создание таблиц по моделям (gORM AutoMigrate)
internal/models/                     модели = таблицы: User, Comet, CometLike
internal/api/                        веб-сервис: api.go (роутинг /api, ошибки), comets.go, users.go,
                                     serializers.go (сериализаторы: структуры запросов и ответов JSON)
internal/repository/                 работа с БД через gORM (удаление на странице ЛР2 — SQL UPDATE)
internal/session/session.go          функция-singleton CurrentUser(): текущий пользователь (константа id = 1)
internal/storage/storage.go          Minio: загрузка и удаление файлов, генерация имён, url файла
internal/handlers/handlers.go        страницы (SSR) из ЛР1–ЛР2
internal/media/media.go              фото и видео по умолчанию для страниц
templates/, static/                  шаблоны и стили страниц
db/seed.sql                          начальные данные (Adminer → «SQL-запрос»)
docs/comets.mdj                      диаграммы StarUML
docs/comets.postman_collection.json  коллекция из 10 запросов для Postman / Insomnia
```

## Запуск

1. `cp .env.example .env` — параметры PostgreSQL (порт 5433) и Minio.
2. `docker compose up -d` — Minio (http://localhost:9001, `minioadmin`), PostgreSQL, Adminer (http://localhost:8082).
3. `go run ./cmd/migrate` — таблицы.
4. Adminer → «SQL-запрос» → `db/seed.sql`: 8 пользователей (пароль у всех `comets2026`),
   10 комет (8 опубликованных, черновик 81P у пользователя 2, удалённая 46P), 31 лайк.
5. `go run ./cmd/comets` → API http://localhost:8080/api/comets, страницы http://localhost:8080/comets.
   `DB_LOG=1` — вывод SQL-запросов в консоль.
6. Postman или Insomnia → Import → `docs/comets.postman_collection.json`
   (в запросе добавления указаны файлы из `media/`, при необходимости выберите их заново).

## База данных

Каскадное удаление не используется: все внешние ключи `ON DELETE RESTRICT`.

**users** — пользователи

| Столбец | Тип | |
|---|---|---|
| id | bigint | первичный ключ |
| login | varchar(50) | уникальный |
| full_name | varchar(100) | |
| password_hash | varchar(100) | bcrypt-хеш пароля, клиенту не отдаётся |

**comets** — услуги (кометы)

| Столбец | Тип | |
|---|---|---|
| id | bigint | первичный ключ |
| comet_name | varchar(100) | наименование |
| comet_description | varchar(500) | краткое описание |
| comet_status | varchar(20) | `draft` / `published` / `deleted` (CHECK) |
| comet_photo | varchar(255) | имя файла изображения в бакете `comets` |
| comet_video | varchar(255) | имя файла видео в бакете `comets` |
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

## API

Ответы — JSON. Ошибки: `{"status": "fail", "message": "…"}` с кодом 400 (неверные данные), 403 (не создатель),
404 (нет записи), 405 (неверный HTTP-метод), 409 (конфликт: черновик уже есть, недопустимая смена статуса, логин занят).
Удалённые кометы клиенту не передаются. Системные поля (`id`, `comet_status`, `creator_id`, `created_at`, `formed_at`,
`comet_photo`, `comet_video`, `likes_count`, `is_mine`, `is_liked`) с клиента передавать нельзя — ответ 400.

### Домен услуг (кометы)

| Метод и URL | Тело запроса | Ответ |
|---|---|---|
| `GET /api/comets?min_orbital_period=&max_orbital_period=` | — | 200, массив опубликованных комет, фильтр по периоду в SQL |
| `GET /api/comets/feed` | — | 200, первая опубликованная комета |
| `GET /api/comets/feed/{id}` | — | 200, комета по id; 404 — удалена или не опубликована |
| `GET /api/comets/feed/{id}?next=true` | — | 200, следующая опубликованная после id (после последней — первая) |
| `GET /api/comets/draft` | — | 200, черновик текущего пользователя (id не указывается); 404 — черновика нет |
| `POST /api/comets` | multipart/form-data: `comet_name`, `comet_photo` (файл jpeg/png/webp/gif до 10 МБ), `comet_video` (файл mp4/webm до 50 МБ) | 201, созданный черновик; 409 — черновик уже есть |
| `PUT /api/comets/{id}/publish` | JSON: `comet_description`, `orbital_period`, `orbit_eccentricity` | 200, опубликованная комета; 409 — не черновик; 403 — чужая |
| `DELETE /api/comets/{id}` | — | 200, `{"id", "comet_status": "deleted"}` — логическое удаление, только своей кометы |
| `POST /api/comets/{id}/like` | JSON: `{"like": 1}` — поставить, `{"like": 0}` — отменить | 200, `{"comet_id", "like", "likes_count"}` |

Комета в ответе:

```json
{
  "id": 1, "comet_name": "1P/Галлея", "comet_description": "…", "comet_status": "published",
  "comet_photo": "halley.jpg", "comet_photo_url": "http://localhost:9000/comets/halley.jpg",
  "comet_video": "halley.mp4", "comet_video_url": "http://localhost:9000/comets/halley.mp4",
  "orbital_period": 75.92, "orbit_eccentricity": 0.968,
  "created_at": "2026-09-01T12:00:00Z", "formed_at": "2026-09-01T18:00:00Z",
  "likes_count": 6, "is_mine": 1, "is_liked": 0
}
```

`is_mine` — 1, если создатель кометы — текущий пользователь; `is_liked` — 1, если текущий пользователь поставил лайк.

Статусы меняются только так: черновик → опубликован (`PUT …/publish`), черновик или опубликован → удалён (`DELETE`).
Вернуть в черновик нельзя. При создании файлы загружаются в Minio с именами вида `comet-11-photo-a1f7ca3f.jpg`
(латиница, id кометы и случайный суффикс). Запись в БД и загрузка выполняются в одной транзакции:
при ошибке загрузки черновик не создаётся, уже загруженные файлы удаляются.

### Домен пользователей

| Метод и URL | Тело запроса | Ответ |
|---|---|---|
| `POST /api/users/register` | JSON: `login` (3–50: латиница, цифры, _), `full_name`, `password` (от 8 символов) | 201, `{"id", "login", "full_name"}`; 409 — логин занят |
| `POST /api/users/login` | — | 200, заглушка до ЛР4 |
| `POST /api/users/logout` | — | 200, заглушка до ЛР4 |

## Страницы (ЛР1–ЛР2)

| Метод и URL | Что делает |
|---|---|
| `GET /comets/feed`, `/comets/feed/{id}`, `?next=true` | лента |
| `GET /comets/draft` | «Добавление»: форма создания или публикации черновика |
| `GET /comets?min_orbital_period=&max_orbital_period=` | плитка с фильтром (слайдеры без JavaScript) |
| `POST /comets/draft`, `POST /comets/draft/publish` | «Далее» и «Опубликовать» (gORM) |
| `POST /comets/{id}/delete` | удаление с плитки — SQL `UPDATE` без ORM |

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

# Comet-Calc-Backend — расстояние кометы от Солнца (ЛР1)

Тема: расстояние кометы от Солнца по её координатам на небесной сфере.
Услуга — **комета**, заявка — **расчёт** расстояния (появится со следующих ЛР).

Поля услуги по теме: период обращения P (лет) и эксцентриситет орбиты e. Фильтрация — по периоду (от и до).

В ЛР1 по заданию только просмотр и поиск: 3 GET-запроса, одна коллекция без БД, без JavaScript.
Заявки и расчёта в ЛР1 нет.

Бэкенд: Go 1.22+ (стандартная библиотека `net/http` + `html/template`, внешних зависимостей нет).

## Структура (MVC)

```
cmd/main.go                    роутинг: 3 GET-запроса + статика, функция шаблона minio
internal/models/comet.go       Model — коллекция услуг models.Comets (без БД)
internal/handlers/handlers.go  Controller — 3 обработчика CometFeed / CometDraft / CometList
templates/comet_feed.html      View — лента
templates/comet_draft.html     View — добавление (черновик)
templates/comet_list.html      View — плитка
templates/partials.html        общий <head> и панель вкладок
static/css/style.css           стили (скопированы с TheSkyLive, список в шапке файла)
docker-compose.yml             Minio + создание бакета comets
media/                         сюда положить изображения и видео перед запуском Minio
```

## Запуск

1. Положите в `media/` файлы с ключами из коллекции (латиница):

   | Комета | Изображение | Видео |
   |---|---|---|
   | 1P/Галлея | halley.jpg | halley.mp4 |
   | 2P/Энке | encke.jpg | encke.mp4 |
   | 67P/Чурюмова — Герасименко | churyumov.jpg | churyumov.mp4 |
   | 12P/Понса — Брукса | pons_brooks.jpg | pons_brooks.mp4 |
   | 109P/Свифта — Туттля | swift_tuttle.jpg | swift_tuttle.mp4 |
   | 9P/Темпеля | tempel1.jpg | tempel1.mp4 |
   | 55P/Темпеля — Туттля | tempel_tuttle.jpg | tempel_tuttle.mp4 |
   | 103P/Хартли | hartley2.jpg | hartley2.mp4 |
   | 81P/Вильда — черновик | wild2.jpg | wild2.mp4 |
   | 46P/Виртанена — удалена | wirtanen.jpg | wirtanen.mp4 |

   Источники и подписи — в разделе «Медиа: источники» ниже.

2. Minio: `docker compose up -d` (консоль http://localhost:9001, логин/пароль `minioadmin`).
   Контейнер `minio-init` создаст бакет `comets`, откроет его на чтение и загрузит файлы из `media/`.
   После изменения файлов в `media/`: `docker compose up --force-recreate minio-init`.

   Образы `minio/minio` и `minio/mc` из методички с конца 2025 года удалены с Docker Hub, поэтому
   используется поддерживаемый сообществом форк `pgsty/minio` + `pgsty/mc` (тот же MinIO, те же команды).
   Настройка вручную, как в методичке (клиент `mc` встроен в контейнер):

   ```
   docker exec -it comets-minio mc alias set local http://localhost:9000 minioadmin minioadmin
   docker exec -it comets-minio mc mb local/comets
   docker exec -it comets-minio mc anonymous set download local/comets
   ```

   Файлы загрузить через консоль http://localhost:9001 → бакет `comets` → Upload.

3. Сервер: `go run ./cmd` → http://localhost:8080/comets
   Другой адрес Minio: `MINIO_URL=http://localhost:9000/comets go run ./cmd`

## Три GET-запроса

| URL | Контроллер | Что делает |
|---|---|---|
| `/comets/feed/` , `/comets/feed/{id}` , `/comets/feed/{id}?next=true` | `CometFeed` | лента: первая комета (из панели вкладок, без ID), комета по ID, следующая после ID |
| `/comets/draft` | `CometDraft` | страница «Добавление» с единственной услугой-черновиком |
| `/comets?min_orbital_period=5&max_orbital_period=40` | `CometList` | плитка всех опубликованных, фильтр на сервере: min_orbital_period ≤ P ≤ max_orbital_period |

Поля кометы: `CometName`, `CometDescription`, `CometPhoto`, `CometVideo`, `CometStatus`, `Likes`;
поля по теме — период обращения `OrbitalPeriod` (лет) и эксцентриситет `OrbitEccentricity` (0 ≤ e < 1).

Фильтрация: два слайдера `input type="range"` «от» и «до» (0–140 лет, шаг 1) с кнопкой «Показать».
Число над ползунком обновляется при перетаскивании без JavaScript — через CSS scroll-driven animations
(`view-timeline` на ползунке, `@property --val`, вывод через `counter()`); в браузерах без их поддержки
показывается значение, пришедшее с сервера. Форма отправляет `GET /comets?min_orbital_period=A&max_orbital_period=B`.
`CometList` переводит каждую границу в float, проходит по `models.PublishedComets()` и отбрасывает кометы
с `OrbitalPeriod < min_orbital_period` или `OrbitalPeriod > max_orbital_period`. Крайние значения (0 и 140),
пустые и некорректные — граница не применяется, без обеих показываются все. Если «от» больше «до»,
границы меняются местами. Значения возвращаются в шаблон (`SliderFrom`, `SliderTo`) и подставляются
в `value` слайдеров, поэтому фильтр сохраняется.
Лайки считаются в контроллере: `LikesCount = len(c.Likes)`.

Статусы: `draft` (только «Добавление»), `published` (лента и плитка), `deleted` (нигде не показывается,
например `/comets/feed/10` → 404).

## Minio в коде

- В коллекции: поля `CometPhoto` и `CometVideo` каждой кометы (`internal/models/comet.go`).
- URL собирает функция шаблона `minio` (`cmd/main.go`): `minio "halley.jpg"` → `http://localhost:9000/comets/halley.jpg`.
- Использование: `comet_feed.html` (video src, poster, миниатюра), `comet_draft.html` (превью фото и видео), `comet_list.html` (фото карточки).

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

// Страница «Как пользоваться» — руководство пользователя AQI Platform.
// Доступна всем авторизованным пользователям.
import styles from './HowToPage.module.css';

// ─── Данные ───────────────────────────────────────────────────────────────────

const AQI_LEVELS = [
  { range: '0–50',    label: 'Хорошее',                     cls: 'aqiGood',      rec: 'Воздух чистый. Идеально для прогулок и занятий спортом на улице.' },
  { range: '51–100',  label: 'Умеренное',                   cls: 'aqiModerate',  rec: 'Воздух приемлемый. Чувствительным людям стоит ограничить длительные нагрузки.' },
  { range: '101–150', label: 'Нездоровое (чувствит.)',      cls: 'aqiUnhealthy', rec: 'Чувствительным группам (астматики, дети, пожилые) стоит сократить время на улице.' },
  { range: '151–200', label: 'Нездоровое',                  cls: 'aqiBad',       rec: 'Всем желательно ограничить пребывание на улице, особенно при физической нагрузке.' },
  { range: '201–300', label: 'Очень нездоровое',            cls: 'aqiVeryBad',   rec: 'Серьёзный риск для здоровья. Рекомендуется находиться в помещении.' },
  { range: '301+',    label: 'Опасное',                     cls: 'aqiHazardous', rec: 'Чрезвычайная ситуация. Все должны избегать пребывания на улице.' },
];

const STEPS_LOGIN = [
  { title: 'Перейдите на сайт', desc: 'Откройте браузер и введите адрес платформы (например: https://aqi.kemerovo.ru).' },
  { title: 'Введите данные', desc: 'На странице входа укажите ваш email и пароль. Пароль выдаётся администратором.' },
  { title: 'Нажмите «Войти»', desc: 'После успешной авторизации откроется главный дашборд с текущими показателями AQI.' },
  { title: 'Сброс пароля', desc: 'Если вы забыли пароль — нажмите «Забыли пароль?» и следуйте инструкции на email.' },
];

const STEPS_DASHBOARD = [
  { title: 'Карточки AQI', desc: 'В верхней части экрана — 4 цветные карточки с текущим индексом качества воздуха по каждому из 4 районов Кемерово.' },
  { title: 'График истории', desc: 'Ниже — график изменения PM2.5 за последние 24 часа. Переключайте период фильтром справа.' },
  { title: 'Таблица датчиков', desc: 'Показывает последние значения всех показателей (NO₂, O₃, CO, SO₂, PM2.5) по каждому датчику.' },
  { title: 'Прогноз на 6 ч', desc: 'Нижняя секция — прогноз AQI по районам на ближайшие 6 часов, рассчитанный алгоритмом EWMA+IDW.' },
];

const STEPS_MAP = [
  { title: 'Откройте карту', desc: 'Нажмите «Карта» в левом меню. Карта загрузится (~1 секунда) и покажет Кемерово с датчиками.' },
  { title: 'Цветные маркеры', desc: 'Каждый маркер соответствует датчику. Цвет отражает категорию AQI: зелёный — хорошее, красный — нездоровое.' },
  { title: 'Кликните на маркер', desc: 'Откроется всплывающее окно с последними значениями: AQI, PM2.5, NO₂, температура, дата измерения.' },
  { title: 'Легенда', desc: 'В правом нижнем углу — цветовая легенда AQI для удобства интерпретации.' },
];

const STEPS_REPORTS = [
  { title: 'Перейдите в Администрирование', desc: 'Доступно только для ролей «Аналитик» и «Администратор».' },
  { title: 'Вкладка «Отчёты»', desc: 'Нажмите вкладку «Отчёты» (иконка 📄).' },
  { title: 'Заполните форму', desc: 'Укажите название отчёта, тип (ежедневный/ежемесячный/произвольный) и период дат.' },
  { title: 'Нажмите «Сгенерировать»', desc: 'Система сформирует CSV-файл. Когда статус изменится на «ready» — кнопка «Скачать» станет активной.' },
];

const ROLES = [
  { name: 'admin', label: 'Администратор', desc: 'Полный доступ: управление пользователями, датчиками, отчётами, просмотр статистики платформы.' },
  { name: 'analyst', label: 'Аналитик', desc: 'Просмотр всех данных, загрузка измерений, генерация и скачивание отчётов.' },
  { name: 'viewer', label: 'Наблюдатель', desc: 'Только просмотр дашборда, карты и прогнозов. Не может загружать данные или создавать отчёты.' },
];

const FAQ = [
  {
    q: 'Данные на дашборде устарели — когда обновятся?',
    a: 'Платформа получает данные от датчиков каждые 5 минут через email (IMAP). Прогноз пересчитывается каждые 20 минут. Обновите страницу (F5) чтобы увидеть свежие данные.',
  },
  {
    q: 'Датчик показывает «Оффлайн» — это проблема?',
    a: 'Датчик считается оффлайн если не присылал данные более 30 минут. Обратитесь к администратору для проверки работоспособности оборудования.',
  },
  {
    q: 'Как встроить виджет на сайт города?',
    a: 'Используйте iframe: <iframe src="https://aqi.kemerovo.ru/widget/" width="400" height="300"></iframe> Виджет работает без авторизации.',
  },
  {
    q: 'Как загрузить данные из файла CSV?',
    a: 'Перейдите в Администрирование → вкладка «Загрузка данных» (для роли Аналитик и выше). Файл должен содержать колонки: sensor_id, time, no2, o3, co, pm25...',
  },
  {
    q: 'Данные хранятся как долго?',
    a: 'По условиям ТЗ — 60 месяцев (5 лет). Данные старше 5 лет удаляются автоматически политикой TimescaleDB retention.',
  },
  {
    q: 'Можно ли получить доступ к API напрямую?',
    a: 'Да. Документация API доступна по адресу /api/v1/docs (Swagger UI). Для доступа потребуется Bearer JWT токен или API-токен (создайте в профиле).',
  },
];

// ─── Компонент ────────────────────────────────────────────────────────────────

function SectionHeader({ icon, title }: { icon: string; title: string }) {
  return (
    <div className={styles.sectionHeader}>
      <span className={styles.sectionIcon}>{icon}</span>
      <h3 className={styles.sectionTitle}>{title}</h3>
    </div>
  );
}

function StepList({ steps }: { steps: { title: string; desc: string }[] }) {
  return (
    <div className={styles.steps}>
      {steps.map((s, i) => (
        <div className={styles.step} key={i}>
          <div className={styles.stepNum}>{i + 1}</div>
          <div className={styles.stepBody}>
            <div className={styles.stepTitle}>{s.title}</div>
            <div className={styles.stepDesc}>{s.desc}</div>
          </div>
        </div>
      ))}
    </div>
  );
}

export default function HowToPage() {
  return (
    <div className={styles.root}>
      <h2 className={styles.pageTitle}>Как пользоваться платформой</h2>

      {/* ── 1. Вход в систему ─────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="🔑" title="1. Вход в систему" />
        <p className={styles.text}>
          Платформа требует авторизации. Ваши учётные данные создаёт{' '}
          <strong>администратор</strong>.
        </p>
        <StepList steps={STEPS_LOGIN} />
        <div className={styles.note}>
          <span className={styles.noteIcon}>💡</span>
          <span>
            Сессия действует <strong>15 минут</strong> (access token). При
            активной работе токен автоматически обновляется. После 30 дней
            бездействия потребуется повторный вход.
          </span>
        </div>
      </section>

      {/* ── 2. Дашборд ───────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="📊" title="2. Дашборд — текущее состояние воздуха" />
        <p className={styles.text}>
          Главная страница платформы. Показывает актуальную информацию о качестве
          воздуха в <strong>4 районах Кемерово</strong>.
        </p>
        <StepList steps={STEPS_DASHBOARD} />
      </section>

      {/* ── 3. Карта ─────────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="🗺️" title="3. Карта датчиков" />
        <p className={styles.text}>
          Интерактивная карта на основе{' '}
          <strong>MapLibre GL</strong> (открытое ПО, лицензия BSD).
        </p>
        <StepList steps={STEPS_MAP} />
      </section>

      {/* ── 4. Что такое AQI ─────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="🌬️" title="4. Шкала AQI — расшифровка значений" />
        <p className={styles.text}>
          <strong>AQI (Air Quality Index)</strong> — числовой индекс качества
          воздуха по методологии US EPA. Рассчитывается на основе концентраций{' '}
          <strong>PM2.5 и NO₂</strong>. Чем ниже — тем чище воздух.
        </p>
        <table className={styles.aqiTable}>
          <thead>
            <tr>
              <th>AQI</th>
              <th>Категория</th>
              <th>Рекомендация</th>
            </tr>
          </thead>
          <tbody>
            {AQI_LEVELS.map((row) => (
              <tr key={row.range}>
                <td>
                  <span className={styles[row.cls as keyof typeof styles]}>{row.range}</span>
                </td>
                <td style={{ fontWeight: 500 }}>{row.label}</td>
                <td style={{ color: 'var(--color-text-muted)', fontSize: 12 }}>{row.rec}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className={styles.note}>
          <span className={styles.noteIcon}>ℹ️</span>
          <span>
            Вещества: <strong>PM2.5</strong> (мелкие твёрдые частицы до 2.5 мкм),{' '}
            <strong>NO₂</strong> (диоксид азота), <strong>SO₂</strong> (диоксид серы),{' '}
            <strong>CO</strong> (угарный газ), <strong>O₃</strong> (озон),{' '}
            <strong>H₂S</strong> (сероводород).
          </span>
        </div>
      </section>

      {/* ── 5. Отчёты ────────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="📄" title="5. Создание отчётов" />
        <p className={styles.text}>
          Отчёты доступны пользователям с ролью <strong>Аналитик</strong> и{' '}
          <strong>Администратор</strong>. Формат — CSV.
        </p>
        <StepList steps={STEPS_REPORTS} />
      </section>

      {/* ── 6. Роли ──────────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="👥" title="6. Роли пользователей" />
        <div className={styles.rolesGrid}>
          {ROLES.map((r) => (
            <div className={styles.roleCard} key={r.name}>
              <div className={styles.roleName}>{r.label}</div>
              <div className={styles.roleDesc}>{r.desc}</div>
            </div>
          ))}
        </div>
      </section>

      {/* ── 7. API доступ ────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="🔌" title="7. Доступ к API" />
        <p className={styles.text}>
          Для интеграции с внешними системами платформа предоставляет{' '}
          <strong>REST API</strong>. Интерактивная документация доступна по адресу{' '}
          <strong>/api/v1/docs</strong> (Swagger UI).
        </p>

        <div className={styles.steps}>
          <div className={styles.step}>
            <div className={styles.stepNum}>1</div>
            <div className={styles.stepBody}>
              <div className={styles.stepTitle}>Создайте API-токен</div>
              <div className={styles.stepDesc}>
                В разделе «Токены» нажмите «+ Создать токен», введите имя.
                Токен показывается <strong>один раз</strong> — сохраните его.
              </div>
            </div>
          </div>
          <div className={styles.step}>
            <div className={styles.stepNum}>2</div>
            <div className={styles.stepBody}>
              <div className={styles.stepTitle}>Используйте в запросах</div>
              <div className={styles.stepDesc}>
                Передавайте токен в заголовке Authorization:
              </div>
            </div>
          </div>
        </div>

        <div className={styles.codeBlock}>{`curl https://aqi.kemerovo.ru/api/v1/measurements/latest \\
  -H "Authorization: Bearer aqi_ваш_токен_здесь"`}</div>

        <div className={styles.note}>
          <span className={styles.noteIcon}>📖</span>
          <span>
            Полная документация API: <strong>/api/v1/docs</strong> — откройте в
            браузере после входа. Там можно протестировать любой эндпоинт
            прямо из браузера.
          </span>
        </div>
      </section>

      {/* ── 8. FAQ ───────────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="❓" title="8. Часто задаваемые вопросы" />
        <div className={styles.faqList}>
          {FAQ.map((item, i) => (
            <div className={styles.faqItem} key={i}>
              <div className={styles.faqQ}>{item.q}</div>
              <div className={styles.faqA}>{item.a}</div>
            </div>
          ))}
        </div>
      </section>

      {/* ── 9. Горячие клавиши ──────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="⌨️" title="9. Быстрые клавиши" />
        <div className={styles.shortcuts}>
          <div className={styles.shortcutRow}>
            <kbd className={styles.kbd}>F5</kbd>
            <span className={styles.text}>Обновить данные на странице</span>
          </div>
          <div className={styles.shortcutRow}>
            <kbd className={styles.kbd}>Esc</kbd>
            <span className={styles.text}>Закрыть форму / попап</span>
          </div>
          <div className={styles.shortcutRow}>
            <kbd className={styles.kbd}>Ctrl + +</kbd>
            <span className={styles.text}>Увеличить карту</span>
          </div>
          <div className={styles.shortcutRow}>
            <kbd className={styles.kbd}>Ctrl + −</kbd>
            <span className={styles.text}>Уменьшить карту</span>
          </div>
        </div>
      </section>

      {/* ── Техподдержка ─────────────────────────────────────────────── */}
      <section className={styles.section}>
        <SectionHeader icon="📞" title="Техническая поддержка" />
        <p className={styles.text}>
          При возникновении проблем обратитесь к администратору системы или
          воспользуйтесь функцией <strong>«Обратная связь»</strong> — кнопка
          в нижней части бокового меню. Мы ответим в рабочее время.
        </p>
        <div className={styles.note}>
          <span className={styles.noteIcon}>🏛️</span>
          <span>
            Система разработана в рамках государственного контракта
            №М05-00285-26-ЭА. Оператор: Администрация города Кемерово.
          </span>
        </div>
      </section>
    </div>
  );
}

// SensorsPage — перечень отечественного оборудования для измерения и передачи данных.
// Справочная страница: газоанализаторы, пылемеры, метеостанции, GSM/4G-модули,
// интерфейсы подключения и операторы связи, сертифицированные в РФ.
import styles from './SensorsPage.module.css';

// ─── Типы ─────────────────────────────────────────────────────────────────────

interface Device {
  name: string;
  model?: string;
  manufacturer: string;
  city: string;
  params: string[];
  interfaces: string[];
  gost?: string;
  note?: string;
  supported: boolean;
}

// ─── Данные ───────────────────────────────────────────────────────────────────

const ANALYZERS: Device[] = [
  {
    name: 'ХОББИТ-Т',
    model: 'SO₂ / NO₂ / CO / H₂S',
    manufacturer: 'ООО «Аналитприбор»',
    city: 'Смоленск',
    params: ['SO₂', 'NO₂', 'CO', 'H₂S'],
    interfaces: ['RS-485 Modbus RTU', '4–20 мА', 'RS-232'],
    gost: 'ГОСТ Р ИСО 16000',
    note: 'Внесён в Госреестр СИ РФ. Поверочный интервал 1 год.',
    supported: true,
  },
  {
    name: 'АМ-0041',
    model: 'Многокомпонентный',
    manufacturer: 'ООО «Аналитприбор»',
    city: 'Смоленск',
    params: ['CO', 'NO', 'NO₂', 'SO₂', 'CO₂', 'O₂'],
    interfaces: ['RS-485 Modbus RTU', 'Ethernet ModbusTCP'],
    gost: 'ГОСТ Р 58759',
    note: 'Электрохимические сенсоры. ПО в комплекте.',
    supported: true,
  },
  {
    name: 'ОПТЭК-1',
    model: 'Лазерный газоанализатор',
    manufacturer: 'ООО «Оптэк»',
    city: 'Санкт-Петербург',
    params: ['CO₂', 'CH₄', 'NH₃', 'H₂O'],
    interfaces: ['RS-232', 'RS-485', 'Ethernet'],
    gost: 'ГОСТ Р 55171',
    note: 'ЛДАС технология. Высокая точность при низких концентрациях.',
    supported: true,
  },
  {
    name: 'ДАГ-500',
    model: 'Фотоионизационный',
    manufacturer: 'ООО «НПП Родник»',
    city: 'Москва',
    params: ['ЛОС (летучие органические соединения)', 'CO'],
    interfaces: ['RS-485 Modbus', '4–20 мА'],
    gost: 'ТУ 26.51.66',
    note: 'Промышленное исполнение IP65. Диапазон -40…+60°C.',
    supported: true,
  },
  {
    name: 'АКЭ-Б',
    model: 'Автоматическая станция',
    manufacturer: 'ООО «ЭКА»',
    city: 'Москва',
    params: ['PM2.5', 'PM10', 'NO₂', 'SO₂', 'CO', 'O₃'],
    interfaces: ['RS-485', 'Ethernet', 'GSM/GPRS'],
    gost: 'ГОСТ Р 56167-2014',
    note: 'Автоматическая станция мониторинга атмосферного воздуха. Метеоблок в комплекте.',
    supported: true,
  },
  {
    name: 'МАК-1',
    model: 'Многокомпонентный',
    manufacturer: 'ЗАО «СВ-Аналитика»',
    city: 'Новосибирск',
    params: ['CO', 'NO₂', 'SO₂', 'O₃', 'NH₃'],
    interfaces: ['RS-485 Modbus RTU', 'RS-232', '4–20 мА'],
    gost: 'ГОСТ Р 55859',
    note: 'Сибирское производство. Адаптирован к суровому климату.',
    supported: true,
  },
  {
    name: 'ХОББИТ-Т-О₃',
    model: 'Анализатор озона',
    manufacturer: 'ООО «Аналитприбор»',
    city: 'Смоленск',
    params: ['O₃'],
    interfaces: ['RS-485 Modbus RTU', '4–20 мА'],
    gost: 'ГОСТ Р ИСО 13964',
    note: 'УФ-фотометрический метод. МВИ аттестована Росстандартом.',
    supported: true,
  },
];

const DUST_SENSORS: Device[] = [
  {
    name: 'ИКП-5М',
    model: 'Импактор-классификатор',
    manufacturer: 'ООО «Оптэк»',
    city: 'Санкт-Петербург',
    params: ['PM1', 'PM2.5', 'PM10', 'ТЧ (общая взвесь)'],
    interfaces: ['RS-485 Modbus', 'Ethernet', 'USB'],
    gost: 'ГОСТ Р ИСО 23210',
    note: 'Лазерный счётчик частиц. Класс защиты IP54.',
    supported: true,
  },
  {
    name: 'ДАП-4',
    model: 'Датчик аэрозольных частиц',
    manufacturer: 'НПП «Кварц»',
    city: 'Екатеринбург',
    params: ['PM2.5', 'PM10'],
    interfaces: ['RS-485 Modbus RTU', '4–20 мА'],
    gost: 'ГОСТ 17.2.6.02',
    note: 'Оптический принцип. Калибровка по ПТФЭ.',
    supported: true,
  },
  {
    name: 'ПМ-01Д',
    model: 'Пылемер промышленный',
    manufacturer: 'НПП «СпецТехника»',
    city: 'Санкт-Петербург',
    params: ['Пыль суммарная', 'PM10'],
    interfaces: ['4–20 мА', 'RS-485'],
    gost: 'ГОСТ Р 55384',
    note: 'Для промышленных зон. Диапазон 0–10 мг/м³.',
    supported: true,
  },
  {
    name: 'Бета-1М',
    model: 'Бета-радиометрический пылемер',
    manufacturer: 'ФГУП «ВНИИМ» им. Менделеева',
    city: 'Санкт-Петербург',
    params: ['PM2.5', 'PM10', 'ТЧ суммарные'],
    interfaces: ['RS-232', 'USB', 'Ethernet'],
    gost: 'ГОСТ Р ИСО 10473',
    note: 'Эталонный метод измерения. Применяется в госсети наблюдений.',
    supported: true,
  },
];

const METEO: Device[] = [
  {
    name: 'АМС-100',
    model: 'Автоматическая метеостанция',
    manufacturer: 'НПП «ЛЭМЗ-Т»',
    city: 'Смоленск',
    params: ['T, °C', 'RH, %', 'P, гПа', 'Скорость ветра', 'Направление ветра', 'Осадки'],
    interfaces: ['RS-485 Modbus', 'RS-232', 'SDI-12', 'GSM/GPRS'],
    gost: 'ГОСТ 33998-2016',
    note: 'Совместима с протоколами Росгидромет. ПО сбора данных в комплекте.',
    supported: true,
  },
  {
    name: 'МС-21',
    model: 'Метеостанция модульная',
    manufacturer: 'ЗАО «Метеоприбор»',
    city: 'Москва',
    params: ['T', 'RH', 'P', 'Ветер', 'Видимость', 'Высота облаков'],
    interfaces: ['RS-485', 'Ethernet', 'MQTT'],
    gost: 'ГОСТ Р 58827',
    note: 'Соответствует требованиям ИКАО для аэродромов.',
    supported: true,
  },
  {
    name: 'ВМС-01',
    model: 'Ветромерная система',
    manufacturer: 'ООО «Вымпел»',
    city: 'Обнинск',
    params: ['Скорость ветра 0–60 м/с', 'Направление ветра 0–360°'],
    interfaces: ['RS-485 Modbus RTU', '4–20 мА', 'частотный выход'],
    gost: 'ГОСТ 7196-94',
    note: 'Ультразвуковой принцип. Без движущихся частей.',
    supported: true,
  },
];

const GSM_MODULES: Device[] = [
  {
    name: 'МТ-301',
    model: '4G LTE роутер',
    manufacturer: 'ООО «М2М-Телематика»',
    city: 'Москва',
    params: ['4G LTE', '3G UMTS', '2G GSM/GPRS'],
    interfaces: ['Ethernet', 'RS-485', 'RS-232', 'WiFi'],
    note: 'Российский производитель. Поддержка SIM МТС/Билайн/МегаФон/Ростелеком/Теле2.',
    supported: true,
  },
  {
    name: 'МТ-201',
    model: 'GSM/GPRS терминал',
    manufacturer: 'ООО «М2М-Телематика»',
    city: 'Москва',
    params: ['2G GSM/GPRS', '3G опционально'],
    interfaces: ['RS-485', 'RS-232', 'Ethernet'],
    note: 'Компактный модуль для датчиков. Поддержка Modbus over TCP.',
    supported: true,
  },
  {
    name: 'CPC509',
    model: 'Промышленный контроллер',
    manufacturer: 'АО «Fastwel»',
    city: 'Москва',
    params: ['4G LTE', 'Ethernet', 'WiFi'],
    interfaces: ['RS-485', 'CAN', 'Ethernet', 'USB'],
    note: 'Отечественный производитель СВТ. Сертификат ФСТЭК.',
    supported: true,
  },
  {
    name: 'TAU-72.IP',
    model: 'Промышленный роутер',
    manufacturer: 'ООО «Элтекс»',
    city: 'Новосибирск',
    params: ['4G LTE', '3G', 'Ethernet', 'WiFi'],
    interfaces: ['LAN/WAN Ethernet', 'RS-232'],
    note: 'Сибирский производитель. Диапазон питания 12–48В DC.',
    supported: true,
  },
  {
    name: 'Teleofis RX108',
    model: 'IoT-роутер',
    manufacturer: 'ООО «Телеофис»',
    city: 'Москва',
    params: ['4G LTE', '3G', '2G'],
    interfaces: ['Ethernet', 'RS-232', 'RS-485'],
    note: 'Разработан в России. Поддержка VPN, MQTT, Modbus TCP/RTU.',
    supported: true,
  },
];

const OPERATORS = [
  { name: 'МТС IoT', plan: 'Тариф «M2M Мониторинг»', bands: '2G/3G/4G/NB-IoT', site: 'mts.ru', note: 'Управление SIM в личном кабинете, статический IP' },
  { name: 'Билайн Business', plan: 'Тариф «М2М Базовый»', bands: '2G/3G/4G', site: 'beeline.ru', note: 'SLA 99.9%, федеральное покрытие' },
  { name: 'МегаФон M2M', plan: 'Тариф «Умное устройство»', bands: '2G/3G/4G/NB-IoT', site: 'megafon.ru', note: 'API управления SIM, групповые тарифы' },
  { name: 'Ростелеком IoT', plan: 'NB-IoT / eMTC', bands: 'NB-IoT/4G', site: 'rt.ru', note: 'Государственный оператор. LoRaWAN в ряде городов' },
  { name: 'Теле2 M2M', plan: 'Тариф «Умные устройства»', bands: '2G/3G/4G', site: 'tele2.ru', note: 'Низкая цена, федеральная лицензия' },
];

const INTERFACES = [
  { name: 'RS-485 / Modbus RTU', desc: 'Основной промышленный стандарт. До 32 устройств на шину, кабель до 1200 м. Поддерживается всеми отечественными анализаторами.' },
  { name: 'ModbusTCP (Ethernet)', desc: 'Modbus по TCP/IP. Интеграция через локальную сеть или VPN. Прямая поддержка платформой.' },
  { name: '4–20 мА (аналог)', desc: 'Токовая петля. Помехозащищённая передача на расстояния до 500 м. Требует АЦП на стороне контроллера.' },
  { name: 'HTTP REST / JSON', desc: 'Прямая отправка измерений на API платформы. Поддерживается GSM-модулями с TCP/IP стеком.' },
  { name: 'MQTT', desc: 'Лёгкий протокол IoT. Поддерживается роутерами Teleofis и Элтекс. В планах — MQTT-брокер в составе платформы.' },
  { name: 'SDI-12', desc: 'Стандарт для метеодатчиков. Поддерживается АМС-100 и другими метеостанциями.' },
  { name: 'Email (IMAP/SMTP)', desc: 'Отправка данных вложением CSV/XLSX на почту. Текущий основной канал по ТЗ. Уже реализован в платформе.' },
];

const GOST_LIST = [
  { code: 'ГОСТ Р 56167-2014', title: 'Выбросы загрязняющих веществ. Автоматические методы измерения' },
  { code: 'ГОСТ 17.2.3.01-86', title: 'Охрана природы. Атмосфера. Правила контроля качества воздуха населённых пунктов' },
  { code: 'ГОСТ Р ИСО 16000-1', title: 'Воздух замкнутых помещений. Часть 1: Общие аспекты' },
  { code: 'ГОСТ Р 55859-2013', title: 'Качество воздуха рабочей зоны. Методы измерения' },
  { code: 'ГН 2.1.6.3492-17', title: 'Предельно допустимые концентрации (ПДК) загрязняющих веществ в атмосферном воздухе' },
  { code: 'РД 52.04.186-89', title: 'Руководство по контролю загрязнения атмосферы (Росгидромет)' },
  { code: 'ГОСТ Р 33998-2016', title: 'Метеорологические приборы и методы наблюдений. АМС' },
];

// ─── Компоненты ───────────────────────────────────────────────────────────────

function DeviceCard({ device }: { device: Device }) {
  return (
    <div className={styles.deviceCard}>
      <div className={styles.deviceHeader}>
        <div>
          <span className={styles.deviceName}>{device.name}</span>
          {device.model && <span className={styles.deviceModel}> — {device.model}</span>}
        </div>
        <span className={device.supported ? styles.badgeSupported : styles.badgePlanned}>
          {device.supported ? 'Поддерживается' : 'В планах'}
        </span>
      </div>
      <div className={styles.deviceMeta}>
        <span className={styles.metaItem}>🏭 {device.manufacturer}</span>
        <span className={styles.metaItem}>📍 {device.city}</span>
        {device.gost && <span className={styles.metaItem}>📋 {device.gost}</span>}
      </div>
      <div className={styles.deviceParams}>
        <strong>Параметры:</strong>{' '}
        {device.params.map(p => (
          <span key={p} className={styles.paramTag}>{p}</span>
        ))}
      </div>
      <div className={styles.deviceInterfaces}>
        <strong>Интерфейсы:</strong>{' '}
        {device.interfaces.map(i => (
          <span key={i} className={styles.ifaceTag}>{i}</span>
        ))}
      </div>
      {device.note && <p className={styles.deviceNote}>{device.note}</p>}
    </div>
  );
}

// ─── Страница ─────────────────────────────────────────────────────────────────

export default function SensorsPage() {
  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>Датчики и оборудование</h1>
      <p className={styles.pageSubtitle}>
        Перечень отечественного оборудования для измерения качества воздуха и
        передачи данных, совместимого с платформой. Всё оборудование произведено
        в России, внесено в Государственный реестр средств измерений РФ.
      </p>

      {/* ── Газоанализаторы ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>🔬</span>
          <div>
            <h2 className={styles.sectionTitle}>Газоанализаторы</h2>
            <p className={styles.sectionDesc}>
              Измерение NO₂, SO₂, CO, O₃, CO₂ и других газовых компонентов
            </p>
          </div>
        </div>
        <div className={styles.deviceGrid}>
          {ANALYZERS.map(d => <DeviceCard key={d.name + d.manufacturer} device={d} />)}
        </div>
      </section>

      {/* ── Пылемеры ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>💨</span>
          <div>
            <h2 className={styles.sectionTitle}>Пылемеры и счётчики частиц</h2>
            <p className={styles.sectionDesc}>
              Измерение взвешенных частиц PM1, PM2.5, PM10
            </p>
          </div>
        </div>
        <div className={styles.deviceGrid}>
          {DUST_SENSORS.map(d => <DeviceCard key={d.name + d.manufacturer} device={d} />)}
        </div>
      </section>

      {/* ── Метеостанции ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>🌤</span>
          <div>
            <h2 className={styles.sectionTitle}>Метеостанции</h2>
            <p className={styles.sectionDesc}>
              Температура, влажность, давление, ветер, осадки
            </p>
          </div>
        </div>
        <div className={styles.deviceGrid}>
          {METEO.map(d => <DeviceCard key={d.name + d.manufacturer} device={d} />)}
        </div>
      </section>

      {/* ── GSM/4G модули ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>📡</span>
          <div>
            <h2 className={styles.sectionTitle}>Модули передачи данных (GSM/3G/4G)</h2>
            <p className={styles.sectionDesc}>
              Отечественные роутеры и GSM-терминалы для передачи данных с датчиков
            </p>
          </div>
        </div>
        <div className={styles.deviceGrid}>
          {GSM_MODULES.map(d => <DeviceCard key={d.name + d.manufacturer} device={d} />)}
        </div>
      </section>

      {/* ── Операторы связи ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>📶</span>
          <div>
            <h2 className={styles.sectionTitle}>Операторы связи для IoT-датчиков</h2>
            <p className={styles.sectionDesc}>
              Российские операторы с тарифами M2M/IoT для датчиков мониторинга
            </p>
          </div>
        </div>
        <div className={styles.operatorTable}>
          <table>
            <thead>
              <tr>
                <th>Оператор</th>
                <th>Тариф</th>
                <th>Стандарты</th>
                <th>Особенности</th>
              </tr>
            </thead>
            <tbody>
              {OPERATORS.map(op => (
                <tr key={op.name}>
                  <td><strong>{op.name}</strong></td>
                  <td>{op.plan}</td>
                  <td><code>{op.bands}</code></td>
                  <td className={styles.textMuted}>{op.note}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* ── Интерфейсы подключения ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>🔌</span>
          <div>
            <h2 className={styles.sectionTitle}>Поддерживаемые интерфейсы подключения</h2>
            <p className={styles.sectionDesc}>
              Протоколы и интерфейсы, через которые платформа принимает данные
            </p>
          </div>
        </div>
        <div className={styles.ifaceList}>
          {INTERFACES.map(i => (
            <div key={i.name} className={styles.ifaceItem}>
              <span className={styles.ifaceName}>{i.name}</span>
              <span className={styles.ifaceDesc}>{i.desc}</span>
            </div>
          ))}
        </div>
      </section>

      {/* ── ГОСТы ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>📋</span>
          <div>
            <h2 className={styles.sectionTitle}>Применимые стандарты и нормативы</h2>
            <p className={styles.sectionDesc}>
              ГОСТ и нормативные документы, которым соответствует платформа
            </p>
          </div>
        </div>
        <div className={styles.gostList}>
          {GOST_LIST.map(g => (
            <div key={g.code} className={styles.gostItem}>
              <span className={styles.gostCode}>{g.code}</span>
              <span className={styles.gostTitle}>{g.title}</span>
            </div>
          ))}
        </div>
      </section>

      {/* ── Добавление нового оборудования ── */}
      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <span className={styles.sectionIcon}>➕</span>
          <div>
            <h2 className={styles.sectionTitle}>Подключение нового оборудования</h2>
          </div>
        </div>
        <p className={styles.text}>
          Платформа принимает данные через <strong>REST API</strong> (POST /api/v1/ingest),{' '}
          <strong>Email IMAP</strong> (CSV/XLSX вложения) или прямой интеграцией
          через <strong>RS-485/Modbus → GSM-роутер → HTTP</strong>.
        </p>
        <p className={styles.text}>
          Для подключения нового типа оборудования обратитесь к администратору
          платформы или изучите раздел{' '}
          <strong>Swagger UI → POST /api/v1/ingest</strong> для формата данных.
        </p>
        <div className={styles.codeBlock}>
          <code>
            {`POST /api/v1/ingest
Authorization: Bearer <api-token>
Content-Type: application/json

{
  "sensor_id": "uuid",
  "measured_at": "2026-01-01T12:00:00Z",
  "pm25": 12.5,
  "no2": 0.04,
  "co": 0.3,
  "so2": 0.01,
  "temperature": -5.2,
  "humidity": 78.0,
  "pressure": 1013.25
}`}
          </code>
        </div>
      </section>
    </div>
  );
}

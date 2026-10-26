// Local drafting only. No network calls, ad account access or publishing.
const fs = require('node:fs');
const path = require('node:path');
const directory = __dirname;
const copy = {
  ru_websites: {
    headlines: ['Сайты для бизнеса', 'Лендинги и сайты компаний', 'Обсудим ваш проект'],
    descriptions: [
      'Структура, дизайн и мобильная версия. Посмотрите работы и опишите задачу.',
      'Формы заявок и запуск сайта. Состав работ, сроки и стоимость согласуем заранее.'
    ]
  },
  ru_software: {
    headlines: ['Программы под ваш бизнес', 'Автоматизация и учёт', 'Обсудим задачи компании'],
    descriptions: [
      'Задачи, сотрудники, топливо и инвентарь в одном сервисе. Обсудим ваш процесс.',
      'Веб-приложение для телефона и компьютера. Размещение и доступы согласуем.'
    ]
  },
  en_websites: {
    headlines: ['Websites for Your Business', 'Landing Pages & Company Sites', 'Discuss Your Website'],
    descriptions: [
      'Design, mobile layouts and enquiry forms. Explore the work and share your brief.',
      'From page structure to launch. Scope, timing and cost agreed before work starts.'
    ]
  },
  en_software: {
    headlines: ['Custom Business Software', 'Automate Everyday Work', 'Built Around Your Workflow'],
    descriptions: [
      'Staff, tasks, fuel and inventory in one web app. Tell me how your business works.',
      'An interface for desktop and mobile. Hosting and data access agreed with you.'
    ]
  }
};

const length = text => [...text].length;
for (const [key, variant] of Object.entries(copy)) {
  if (variant.headlines.length !== 3 || variant.descriptions.length !== 2) throw new Error(`Invalid asset count: ${key}`);
  for (const text of variant.headlines) {
    if (length(text) > 30 || text.split(/\s+/).some(word => length(word) > 22)) throw new Error(`Headline too long: ${key}: ${text}`);
  }
  for (const text of variant.descriptions) {
    if (length(text) > 81 || text.split(/\s+/).some(word => length(word) > 23)) throw new Error(`Description too long: ${key}: ${text}`);
    if ((text.match(/\p{P}/gu) || []).length > 15) throw new Error(`Too much punctuation: ${key}`);
  }
}

function csv(name, rows) {
  const keys = Object.keys(rows[0]);
  const encode = value => '"' + String(value ?? '').replaceAll('"', '""') + '"';
  const output = [keys, ...rows.map(row => keys.map(key => row[key]))].map(row => row.map(encode).join(',')).join('\r\n');
  fs.writeFileSync(path.join(directory, name), '\uFEFF' + output + '\r\n', 'utf8');
}

const campaigns = [];
for (const country of ['BY', 'RU', 'PL']) {
  const language = country === 'PL' ? 'en' : 'ru';
  const platform = country === 'PL' ? 'google_search_candidate' : 'yandex_search_candidate';
  for (const service of ['websites', 'software']) {
    const variant = copy[`${language}_${service}`];
    const campaign = `qoncept_${country.toLowerCase()}_${service}_${language}_test`;
    const landing = `https://qoncept.by/${language === 'en' ? 'en/' : ''}${service}`;
    const source = country === 'PL' ? 'google' : 'yandex';
    campaigns.push({
      draft_id: campaign,
      status: 'LOCAL_DRAFT_NOT_PUBLISHED',
      platform_candidate: platform,
      country_candidate: country,
      language,
      service,
      phase: country === 'BY' && service === 'websites' ? 'FIRST_CANDIDATE_ONLY' : 'LATER_REVIEW',
      daily_budget: '',
      allocated_budget: '',
      final_url: landing,
      tracking_url_draft: `${landing}?utm_source=${source}&utm_medium=cpc&utm_campaign=${campaign}`,
      headline_1: variant.headlines[0], headline_1_length: length(variant.headlines[0]),
      headline_2: variant.headlines[1], headline_2_length: length(variant.headlines[1]),
      headline_3: variant.headlines[2], headline_3_length: length(variant.headlines[2]),
      description_1: variant.descriptions[0], description_1_length: length(variant.descriptions[0]),
      description_2: variant.descriptions[1], description_2_length: length(variant.descriptions[1])
    });
  }
}
csv('campaign-drafts.csv', campaigns);

const seeds = {
  ru_websites: [
    'заказать сайт', 'создание сайта для бизнеса', 'разработка корпоративного сайта',
    'заказать лендинг', 'разработать сайт компании', 'стоимость разработки сайта',
    'разработка сайта под ключ', 'разработка сайта визитки', 'создание сайта для услуг'
  ],
  ru_software: [
    'разработка программного обеспечения на заказ', 'заказать программу для бизнеса',
    'разработка веб приложения', 'автоматизация бизнес процессов', 'разработка внутреннего сервиса',
    'система учета топлива на заказ', 'система учета инвентаря на заказ',
    'разработка системы учета', 'веб приложение для сотрудников', 'программа учета инструмента на заказ'
  ],
  en_websites: [
    'business website development', 'custom website development', 'company website design',
    'landing page development', 'hire website developer', 'website development quote',
    'small business website developer', 'corporate website development'
  ],
  en_software: [
    'custom business software development', 'business process automation development',
    'custom web application development', 'internal business software development',
    'inventory tracking software development', 'fuel tracking software development',
    'bespoke business software', 'custom employee management software', 'workflow automation developer'
  ]
};
const keywords = [];
for (const campaign of campaigns) {
  for (const keyword of seeds[`${campaign.language}_${campaign.service}`]) keywords.push({
    draft_id: campaign.draft_id, country_candidate: campaign.country_candidate,
    language: campaign.language, service: campaign.service, keyword,
    intent: 'COMMERCIAL_CANDIDATE_NOT_VOLUME_VERIFIED',
    google_match_candidate: 'EXACT_OR_PHRASE_REVIEW',
    yandex_match_candidate: 'REVIEW_OPERATORS_IN_PLATFORM',
    status: 'LOCAL_DRAFT_NOT_PUBLISHED'
  });
}
csv('keyword-candidates.csv', keywords);

const negatives = {
  ru_common: ['вакансия', 'вакансии', 'стажировка', 'курсы программирования', 'курсовая работа', 'дипломная работа', 'скачать бесплатно', 'своими руками', 'уроки программирования'],
  ru_websites: ['бесплатный шаблон сайта', 'бесплатный конструктор сайтов', 'создать сайт самому', 'как создать сайт', 'учебник html'],
  ru_software: ['скачать программу', 'скачать приложение', 'ключ активации', 'кряк', 'лабораторная работа', 'решить задачу по программированию'],
  en_common: ['developer jobs', 'job vacancies', 'internship', 'programming courses', 'programming tutorial', 'free download', 'cracked software', 'source code download'],
  en_websites: ['free website builder', 'free website template', 'how to build a website', 'wordpress tutorial', 'html course'],
  en_software: ['software download', 'license key', 'activation key', 'programming assignment', 'homework help', 'free app download']
};
const negativeRows = [];
for (const [group, terms] of Object.entries(negatives)) {
  const [language, service] = group.split('_');
  for (const term of terms) negativeRows.push({
    language, service_scope: service === 'common' ? 'websites|software' : service,
    countries_candidate: language === 'ru' ? 'BY|RU' : 'PL', negative_phrase: term,
    google_match_candidate: 'NEGATIVE_PHRASE_REVIEW',
    yandex_match_candidate: 'REVIEW_AS_NEGATIVE_PHRASE',
    rationale: service === 'common' ? 'EDUCATION_JOBS_OR_FREE_DOWNLOAD' : 'DIY_DOWNLOAD_OR_STUDY_INTENT',
    status: 'LOCAL_DRAFT_NOT_PUBLISHED'
  });
}
csv('negative-keyword-candidates.csv', negativeRows);

const sections = ['# Тексты поисковых объявлений', '', 'Локальные варианты для согласования. Ничего не опубликовано. Во всех вариантах 3 заголовка и 2 описания; длина включает пробелы и знаки препинания.', ''];
for (const [key, variant] of Object.entries(copy)) {
  sections.push(`## ${key}`, '', '| Поле | Текст | Символов |', '|---|---|---:|');
  variant.headlines.forEach((text, index) => sections.push(`| Заголовок ${index + 1} | ${text.replaceAll('|', '\\|')} | ${length(text)} |`));
  variant.descriptions.forEach((text, index) => sections.push(`| Описание ${index + 1} | ${text.replaceAll('|', '\\|')} | ${length(text)} |`));
  sections.push('');
}
sections.push('Все заголовки укладываются в 30 символов Google и 56 Яндекса. Описания намеренно ограничены 81 символом: подходят также под предел Google 90. Для Яндекса проверены длина слова и число знаков препинания. Финальная модерация площадкой не выполнялась.', '');
fs.writeFileSync(path.join(directory, 'ad-copy.md'), sections.join('\n'), 'utf8');
console.log(JSON.stringify({campaigns: campaigns.length, keywordCandidates: keywords.length, negativeCandidates: negativeRows.length, copyVariants: Object.keys(copy).length, checks: 'PASS'}, null, 2));

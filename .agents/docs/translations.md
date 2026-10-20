# Translations

User-facing strings must be translatable.

- Frontend: source strings in `frontend/src/i18n/lang/en.json`. In components: `const {t} = useI18n({useScope: 'global'})`, then `t('your.key')`.
- API (mostly notifications): source strings in `pkg/i18n/lang/en.json`. Use `i18n.T(lang, "your.key", params...)`.
- This fork maintains Russian itself: add every new key to both `en.json` and `ru-RU.json`. Other languages come from upstream releases; leave them alone.
- Before adding a new string, check if a string with the same value already exists. If it does, use the existing key.

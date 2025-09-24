package common

import "net/http"

func LangCodeName(langCode string) (column string, err error) {
	switch langCode {
	case "sv":
		column = "Swedish"
	case "en":
		column = "English"
	case "ar":
		column = "Arabic"
	case "ckb":
		column = "Sorani"
	case "es":
		column = "Spanish"
	case "fa":
		column = "Farsi"
	case "fa-AF":
		column = "Dari"
	case "fr":
		column = "French"
	case "pl":
		column = "Polish"
	case "ru":
		column = "Russian"
	case "so":
		column = "Somali"
	case "th":
		column = "Thai"
	case "ti":
		column = "Tigrinya"
	case "tr":
		column = "Turkish"
	case "uk":
		column = "Ukrainian"
	case "ro":
		column = "Romanian"
	case "bs":
		column = "Bosnian"
	case "vi":
		column = "Vietnamese"
	case "sq":
		column = "Albanian"
	default:
		err = NewError(http.StatusBadRequest).Msg("langCode not valid").Str("langCode", langCode)
	}
	return
}

func LangCode(lang string) string {
	switch lang {
	case "Swedish":
		return "sv"
	case "English":
		return "en"
	case "Arabic":
		return "ar"
	case "Spanish":
		return "es"
	case "Farsi":
		return "fa"
	case "Dari":
		return "fa-AF"
	case "French":
		return "fr"
	case "Polish":
		return "pl"
	case "Russian":
		return "ru"
	case "Somali":
		return "so"
	case "Sorani":
		return "ckb"
	case "Thai":
		return "th"
	case "Tigrinya":
		return "ti"
	case "Turkish":
		return "tr"
	case "Ukrainian":
		return "uk"
	case "Romanian":
		return "ro"
	case "Bosnian":
		return "bs"
	case "Vietnamese":
		return "vi"
	case "Albanian":
		return "sq"
	default:
		panic("lang not valid: " + lang)
	}
}

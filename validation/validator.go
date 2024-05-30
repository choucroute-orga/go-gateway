package validation

import (
	"gateway/configuration"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type Validation struct {
	Validate *validator.Validate
	Trans    ut.Translator
}

func New(conf *configuration.Configuration) *Validation {

	var trans ut.Translator
	validate := validator.New()

	if conf.TranslateValidation {
		en := en.New()
		uni := ut.New(en, en)
		trans, _ = uni.GetTranslator("en")
		en_translations.RegisterDefaultTranslations(validate, trans)
	}

	return &Validation{
		Validate: validate,
		Trans:    trans,
	}
}

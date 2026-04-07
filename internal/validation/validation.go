package validation

import (
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"regexp"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	enLocales "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"go.uber.org/zap"
)

var translator ut.Translator

func addCustomTag(v *validator.Validate, tag string, validate func(field string) bool, translation string) {
	log := zap.L()

	err := v.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		fieldStr := fl.Field().String()
		return validate(fieldStr)
	})
	if err != nil {
		log.Fatal(
			"Failed to register custom tag validation",
			zap.Error(err),
			zap.String("tag", tag),
		)
	}

	err = v.RegisterTranslation(tag, translator, func(ut ut.Translator) error {
		return ut.Add(tag, translation, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field())
		return t
	})
	if err != nil {
		log.Fatal(
			"Failed to register custom tag translation",
			zap.Error(err),
			zap.String("tag", tag),
		)
	}
}

func InitValidator() {
	log := zap.L()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		en := enLocales.New()
		uni := ut.New(en, en)

		translator, _ = uni.GetTranslator("en")
		err := enTranslations.RegisterDefaultTranslations(v, translator)
		if err != nil {
			log.Fatal(
				"Failed to register default translations",
				zap.Error(err),
			)
		}

		addCustomTag(v, "displayname", func(displayName string) bool {
			re := regexp.MustCompile(`^[a-zA-Z0-9 _\-.]+$`)
			return re.MatchString(displayName)
		}, "{0} can contain only a-Z, 0-9, whitespace, dot, underscore and dash characters.")

		addCustomTag(v, "ricetitle", func(riceTitle string) bool {
			re := regexp.MustCompile(`^[\[\]()a-zA-Z0-9 '_-]+$`)
			return re.MatchString(riceTitle)
		}, "{0} can contain only a-Z, 0-9, -, _, [], () and whitespace characters.")

		log.Info("Validator initialized")
	}
}

func checkValidationErrors(err error) errs.AppError {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		translated := slices.Collect(maps.Values(ve.Translate(translator)))
		return errs.UserErrors(translated, http.StatusBadRequest)
	} else if errors.Is(err, io.EOF) {
		return errs.UserError("Request body is required", http.StatusBadRequest)
	}

	return errs.UserError("Failed to parse and decode request body", http.StatusBadRequest)
}

func ValidateJSON(c *gin.Context, obj any) errs.AppError {
	if err := c.ShouldBindJSON(obj); err != nil {
		return checkValidationErrors(err)
	}
	return nil
}

func ValidateForm(c *gin.Context, obj any) error {
	if err := c.ShouldBind(obj); err != nil {
		return checkValidationErrors(err)
	}
	return nil
}

var openFailed = errs.UserError("Couldn't open and read the uploaded file", http.StatusUnprocessableEntity)

func ValidateFileAsImage(formFile *multipart.FileHeader) (string, errs.AppError) {
	// file, err := formFile.Open()
	// if err != nil {
	// 	return "", openFailed
	// }

	// mtype, _ := mimetype.DetectReader(file)
	// log.Println(mtype)
	// if !mtype.Is("image/jpeg") && !mtype.Is("image/png") {
	// 	return "", errs.UserError("Unsupported file type! Only png/jpeg is accepted", http.StatusUnsupportedMediaType)
	// }

	// return mtype.Extension(), nil
	name := strings.ToLower(formFile.Filename)
	if strings.HasSuffix(name, ".png") {
		return ".png", nil
	} else if strings.HasSuffix(name, ".jpg") {
		return ".jpg", nil
	} else {
		return "", errs.UserError("Unsupported file type! Only png/jpeg is accepted", http.StatusUnsupportedMediaType)
	}
}

type opener interface {
	Open() (multipart.File, error)
}

// validateArchive checks whether provided file (from opener interface) is a valid archive.
// It's using a custom interface so it can be unit tested :p
func validateArchive(o opener) (string, errs.AppError) {
	file, err := o.Open()
	if err != nil {
		return "", openFailed
	}
	defer func() {
		if err := file.Close(); err != nil {
			zap.L().Error(
				"Failed to close archive file during validation",
				zap.Error(err),
			)
		}
	}()

	mtype, _ := mimetype.DetectReader(file)
	if !mtype.Is("application/zip") {
		return "", errs.UserError(
			"Unsupported file type! Only zip is accepted",
			http.StatusUnsupportedMediaType,
		)
	}

	return mtype.Extension(), nil
}

func ValidateFileAsArchive(formFile *multipart.FileHeader) (string, errs.AppError) {
	return validateArchive(formFile)
}

// case-insensitive version of strings.Contains
func ContainsBlacklistedWord(text string, blacklist []string) bool {
	text = strings.ToLower(text)

	for _, word := range blacklist {
		word = strings.ToLower(word)

		// check for whole words and not substrings
		pattern := `\b` + regexp.QuoteMeta(word) + `\b`
		re := regexp.MustCompile(pattern)

		if re.MatchString(text) {
			return true
		}
	}

	return false
}

func IsUsernameBlacklisted(username string) bool {
	bl := config.Config.Blacklist

	// check for exact matches
	exact := append(bl.DisplayNames, bl.Usernames...)
	for _, word := range exact {
		if strings.EqualFold(word, username) {
			return true
		}
	}

	return ContainsBlacklistedWord(username, bl.Words)
}

func IsDisplayNameBlacklisted(displayName string) bool {
	bl := config.Config.Blacklist
	contains := append(bl.Words, bl.DisplayNames...)
	return ContainsBlacklistedWord(displayName, contains)
}

package inline

import (
	"github.com/metafates/mangal/constant"
	"github.com/metafates/mangal/downloader"
	"github.com/metafates/mangal/log"
	"github.com/metafates/mangal/source"
	"github.com/spf13/viper"
	"os"
)

func Run(options *Options) (err error) {
	if options.Out == nil {
		options.Out = os.Stdout
	}

	var mangas []*source.Manga
	for _, src := range options.Sources {
		m, err := src.Search(options.Query)
		if err != nil {
			return err
		}

		mangas = append(mangas, m...)
	}

	if options.MangaPicker.IsAbsent() && options.ChaptersFilter.IsAbsent() {
		if viper.GetBool(constant.MetadataFetchAnilist) {
			for _, manga := range mangas {
				_ = manga.PopulateMetadata(func(string) {})
			}
		}

		marshalled, err := asJson(mangas, options)
		if err != nil {
			return err
		}

		_, err = options.Out.Write(marshalled)
		return err
	}

	// manga picker can only be none if json is set
	if options.MangaPicker.IsAbsent() {
		// preload all chapters
		for _, manga := range mangas {
			if err = prepareManga(manga, options); err != nil {
				return err
			}
		}

		marshalled, err := asJson(mangas, options)
		if err != nil {
			return err
		}

		_, err = options.Out.Write(marshalled)
		return err
	}

	var chapters []*source.Chapter

	if len(mangas) == 0 {
		return nil
	}

	manga := options.MangaPicker.MustGet()(mangas)
	chapters, err = manga.Source.ChaptersOf(manga)
	if err != nil {
		return err
	}

	if options.ChaptersFilter.IsPresent() {
		chapters, err = options.ChaptersFilter.MustGet()(chapters)
		if err != nil {
			return err
		}
	}

	if options.Json {
		if err = prepareManga(manga, options); err != nil {
			return err
		}

		marshalled, err := asJson([]*source.Manga{manga}, options)
		if err != nil {
			return err
		}

		_, err = options.Out.Write(marshalled)
		return err
	}

	for _, chapter := range chapters {
		if options.Download {
			path, err := downloader.Download(chapter, func(string) {})
			if err != nil && viper.GetBool(constant.DownloaderStopOnError) {
				return err
			}

			_, err = options.Out.Write([]byte(path + "\n"))
			if err != nil {
				log.Warn(err)
			}
		} else {
			err := downloader.Read(chapter, func(string) {})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

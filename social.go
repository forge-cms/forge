package forge

// — SocialFeature —————————————————————————————————————————————————————————

// SocialFeature selects which social sharing meta tags forge:head emits for
// a module. Use the predefined constants [OpenGraph] and [TwitterCard].
type SocialFeature int

const (
	// OpenGraph enables Open Graph meta tags (og:title, og:description,
	// og:image, og:type, og:url, and article:* for Article content).
	OpenGraph SocialFeature = 1

	// TwitterCard enables Twitter Card meta tags (twitter:card, twitter:title,
	// twitter:description, twitter:image, twitter:creator).
	TwitterCard SocialFeature = 2
)

// socialOption carries the SocialFeature flags for a module.
// Created by [Social]; applied in NewModule.
type socialOption struct{ features []SocialFeature }

func (socialOption) isOption() {}

// Social returns an [Option] that documents which social sharing tag sets a
// module emits. The forge:head partial always renders Open Graph and Twitter
// Card tags when [Head.Title] is non-empty — Social() is declarative metadata
// that makes intent explicit at the call site.
//
//	app.Content(&BlogPost{},
//	    forge.At("/posts"),
//	    forge.Social(forge.OpenGraph, forge.TwitterCard),
//	)
//
// To customise per-item Twitter output, set [Head.Social] on the content type's
// Head() method:
//
//	func (p *BlogPost) Head() forge.Head {
//	    return forge.Head{
//	        // ...
//	        Social: forge.SocialOverrides{
//	            Twitter: forge.TwitterMeta{
//	                Card:    forge.SummaryLargeImage,
//	                Creator: "@alice",
//	            },
//	        },
//	    }
//	}
func Social(features ...SocialFeature) Option { return socialOption{features: features} }

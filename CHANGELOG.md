# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project adheres to
[Semantic Versioning]. Numbers in parentheses are related issues or pull
requests.

## [Unreleased]

## [0.5.0] - 2024-11-29

### Fixed

- Note for identifier.json creation task ([#74])
- Born Digital AIPs identification ([#79] and [#90])
- Use relative paths in metadata validation errors ([#87])

### Added

- PREMIS creation task to preprocessing workflow result ([#73])
- Support for SHA-1, SHA-256, and SHA-512 checksums in manifest ([#88])

## [0.4.0] - 2024-11-15

### Changed

- Metadata files are combined into one in the AIS workflow ([#77])
- AIS workflow is started by Enduro instead of AMSS and the API server ([#78])

### Removed

- API server used for AIS integration AMSS post-store callback ([#78])

## [0.3.0] - 2024-10-29

### Changed

- Don't rename Prozess_Digitalisierung_PREMIS.xml ([#58])
- Use relative paths in error messages ([#59])

## [0.2.0] - 2024-10-23

### Changed

- Use xmllint to validate SIP manifests ([#39])
- Read allowed file formats from a CSV file ([#60])

### Added

- [xmllint](https://linux.die.net/man/1/xmllint) dependency ([#39])

### Removed

- Python and lxml dependency ([#39])

## [0.1.0] - 2024-09-19

Initial release.

[unreleased]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/artefactual-sdps/preprocessing-sfa/releases/tag/v0.1.0
[#90]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/90
[#88]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/88
[#87]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/87
[#79]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/79
[#78]: https://github.com/artefactual-sdps/preprocessing-sfa/pull/78
[#77]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/77
[#74]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/74
[#73]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/73
[#60]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/60
[#59]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/59
[#58]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/58
[#39]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/39
[keep a changelog]: https://keepachangelog.com/en/1.1.0
[semantic versioning]: https://semver.org/spec/v2.0.0.html

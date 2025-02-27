# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project adheres to
[Semantic Versioning]. Numbers in parentheses are related issues or pull
requests.

## [Unreleased]

## [0.10.0] - 2025-02-27

### Added

- Add PREMIS event and agent for file validation ([#118])

## [0.9.0] - 2025-02-13

### Changed

- Improve empty directory error message ([#108])
- Improve invalid character error message ([#109])

## [0.8.0] - 2025-01-29

### Added

- Fail structure validation if any directory is empty ([#108])
- Detect invalid characters in file/dir names ([#109])

## [0.7.0] - 2025-01-17

### Changed

- Expect compressed SIPs from Enduro parent workflow ([#106])

### Added

- Persistence package for MySQL database ([#106])
- Check if SIP has already been ingested ([#106])

## [0.6.0] - 2025-01-09

### Added

- veraPDF validation of PDF/A files ([#63])
- BagIt SIPs support and validation ([#93], [#94] and [#97])
- Logical metadata PREMIS file support and validation ([#98])

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

[unreleased]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.10.0...HEAD
[0.10.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/artefactual-sdps/preprocessing-sfa/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/artefactual-sdps/preprocessing-sfa/releases/tag/v0.1.0
[#118]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/118
[#109]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/109
[#108]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/108
[#106]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/106
[#98]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/98
[#97]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/97
[#94]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/94
[#93]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/93
[#90]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/90
[#88]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/88
[#87]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/87
[#79]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/79
[#78]: https://github.com/artefactual-sdps/preprocessing-sfa/pull/78
[#77]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/77
[#74]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/74
[#73]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/73
[#63]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/63
[#60]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/60
[#59]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/59
[#58]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/58
[#39]: https://github.com/artefactual-sdps/preprocessing-sfa/issues/39
[keep a changelog]: https://keepachangelog.com/en/1.1.0
[semantic versioning]: https://semver.org/spec/v2.0.0.html

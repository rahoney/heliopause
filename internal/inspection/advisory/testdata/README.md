# Frozen OSV record facts

`ghsa-462w-v97r-4m45.json` projects the public record acquired from
[OSV](https://api.osv.dev/v1/vulns/GHSA-462w-v97r-4m45) on 2026-11-03.
The underlying record is from the [GitHub Advisory Database](https://github.com/github/advisory-database),
licensed under [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/).
The projection retains identity, modified time, aliases, severity and affected
package/version facts; descriptive text, references and database-specific fields
were removed. Original response SHA256:
`aa699023804f4a5198a77e29158594b335b82f2e1c227788832fa5c22a9bb42c`.

This fixture tests one historical advisory. Its fixed boundary does not establish
that a Jinja2 version is free of other advisories or safe to install today.

`ghsa-cpwx-vrp4-4pq7.json`은2026-11-03 고정 OSV API에서 취득한
[GHSA-cpwx-vrp4-4pq7](https://api.osv.dev/v1/vulns/GHSA-cpwx-vrp4-4pq7)의
동일 필드 projection이다. 원본2635B의 SHA-256은
`343a0678ec31607984c5ad8c386e4bd5395ce21de58a0361c73f8a4830766349`다.
GitHub Advisory Database의 CC-BY4.0 출처·license는 위와 같다.
Batch의 modified는 `.526758Z`, 상세 record는 `.526758681Z`여서
provider 저장소의 microsecond index와 원본 nanosecond record 대조를 재현한다.
원래 modified·ID·package/version·severity를 변경하지 않았다.

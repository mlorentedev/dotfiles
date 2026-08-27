# Changelog

## [0.51.0](https://github.com/mlorentedev/dotfiles/compare/v0.50.0...v0.51.0) (2026-08-27)


### Features

* **cli:** add orca ADE keybindings deployment and bidirectional settings capture ([#1274](https://github.com/mlorentedev/dotfiles/issues/1274)) ([da47c4e](https://github.com/mlorentedev/dotfiles/commit/da47c4ec6f203d18f49565236ea0e7b5e8a9b124))
* **cli:** cut over knowledge-crystallize.{sh,ps1} to dotf vault crystallize ([#1276](https://github.com/mlorentedev/dotfiles/issues/1276)) ([0df4bf8](https://github.com/mlorentedev/dotfiles/commit/0df4bf8d59a012b226ec214ecbf3640e6ab47c7a)), closes [#1269](https://github.com/mlorentedev/dotfiles/issues/1269)
* **harness:** add cyclomatic complexity skill and harness evaluation benchmarks ([#1246](https://github.com/mlorentedev/dotfiles/issues/1246)) ([d9a6e3a](https://github.com/mlorentedev/dotfiles/commit/d9a6e3aaed3a1182833f6e2d182ddcb52a012054))
* **harness:** declare gemini pool for agy in model map to enable reviewer fallback ([#1264](https://github.com/mlorentedev/dotfiles/issues/1264)) ([86a1763](https://github.com/mlorentedev/dotfiles/commit/86a1763975118296df54574aa8ceffdd6402762a))
* **harness:** declare where model ids are pinned, and check they resolve ([#1256](https://github.com/mlorentedev/dotfiles/issues/1256)) ([d7e5ddc](https://github.com/mlorentedev/dotfiles/commit/d7e5ddcc21d4988aef49c058ef31dbb2bd35ed0b))
* **harness:** the agnostic binding core for personas, and emission that coexists with a live third-party writer ([#1272](https://github.com/mlorentedev/dotfiles/issues/1272)) ([19618e2](https://github.com/mlorentedev/dotfiles/commit/19618e25912f07ce7a128fa872b4b18d08c31e00))
* **mem:** a thread is the branch, so work follows you between machines ([#1280](https://github.com/mlorentedev/dotfiles/issues/1280)) ([4397ba3](https://github.com/mlorentedev/dotfiles/commit/4397ba3ea8977ce17a04022145f7e99e99971f60))
* **mem:** one handoff thread per worktree, so concurrent sessions stop clobbering each other ([#1279](https://github.com/mlorentedev/dotfiles/issues/1279)) ([71a4c39](https://github.com/mlorentedev/dotfiles/commit/71a4c39f7e77f369a7a58be849948c0a41d066f3))
* **pi:** add qwen3.8-flash and glm5.3-flash to the NaN catalog ([#1255](https://github.com/mlorentedev/dotfiles/issues/1255)) ([cba6be2](https://github.com/mlorentedev/dotfiles/commit/cba6be2f0290ae84727b586acfef6c99be821550))
* **pi:** sync enabledModels into an existing settings.json on setup ([#1259](https://github.com/mlorentedev/dotfiles/issues/1259)) ([661b9f8](https://github.com/mlorentedev/dotfiles/commit/661b9f8f5a3d1c7d4f5d8d3fe77f606583725058))


### Bug Fixes

* **cli:** stamp spec init created field with local calendar date ([#1257](https://github.com/mlorentedev/dotfiles/issues/1257)) ([5852cf2](https://github.com/mlorentedev/dotfiles/commit/5852cf20c719fa23ff0234d7ec87739f626854c2))
* **harness:** apply the reviewer's findings on the persona gate ([#1275](https://github.com/mlorentedev/dotfiles/issues/1275)) ([a18981e](https://github.com/mlorentedev/dotfiles/commit/a18981e2d1b6c3a961843154f50eb2a48d6a1ec6))
* **pi:** a hand-wired extension symlink shadowed the packaged one, and pi would not start ([#1248](https://github.com/mlorentedev/dotfiles/issues/1248)) ([5f80af8](https://github.com/mlorentedev/dotfiles/commit/5f80af892edb543410554b684e9366a823262f65))

## [0.50.0](https://github.com/mlorentedev/dotfiles/compare/v0.49.0...v0.50.0) (2026-08-26)


### Features

* **agents:** fan out the invocable roster — five personas, the steward catalog entry, and a drift guard ([#1240](https://github.com/mlorentedev/dotfiles/issues/1240)) ([2729c09](https://github.com/mlorentedev/dotfiles/commit/2729c09ce7e060a30d54e0f7c27d73414abe3ca5))


### Bug Fixes

* **cli-042:** the post-deploy check reported FAIL when healthy and SKIP when dead ([#1235](https://github.com/mlorentedev/dotfiles/issues/1235)) ([d49e6f8](https://github.com/mlorentedev/dotfiles/commit/d49e6f87357483cb5cc3b87ca5a0b400527ceb4e))
* **opencode:** drop the ollama provider, which had stopped opencode from starting ([#1242](https://github.com/mlorentedev/dotfiles/issues/1242)) ([f2a2d77](https://github.com/mlorentedev/dotfiles/commit/f2a2d77501fbca8413c01fe72160d0124cf3e607))

## [0.49.0](https://github.com/mlorentedev/dotfiles/compare/v0.48.1...v0.49.0) (2026-08-25)


### Features

* **agent:** give hive's daemon its worker contract without a credential on disk ([#1230](https://github.com/mlorentedev/dotfiles/issues/1230)) ([981fd93](https://github.com/mlorentedev/dotfiles/commit/981fd9353afbd9d446bcff7812142ae6d1a1cfc0))
* **cli:** bound dispatch concurrency and enforce the per-dispatch deadline ([#1212](https://github.com/mlorentedev/dotfiles/issues/1212)) ([8171ac1](https://github.com/mlorentedev/dotfiles/commit/8171ac10badd8e1a92d059ece91e94426d7fd699)), closes [#1190](https://github.com/mlorentedev/dotfiles/issues/1190)
* **cli:** deny dispatch on a machine that has not declared who it is ([#1213](https://github.com/mlorentedev/dotfiles/issues/1213)) ([9531527](https://github.com/mlorentedev/dotfiles/commit/95315274eb994d64a4741a94e0d1366f02f1d05f))
* **cli:** dotf agent run dispatches over the tier chain ([#1209](https://github.com/mlorentedev/dotfiles/issues/1209)) ([7e734c4](https://github.com/mlorentedev/dotfiles/commit/7e734c4de291c306cdfde6fdbf96bc271224f925)), closes [#1190](https://github.com/mlorentedev/dotfiles/issues/1190)
* **cli:** probe real backends and route each chain entry to one that serves it ([#1227](https://github.com/mlorentedev/dotfiles/issues/1227)) ([4783a2d](https://github.com/mlorentedev/dotfiles/commit/4783a2dc316a377f06b5d9ab13d8893b8bff7ee6))
* **pi:** declare pi packages in a manifest setup reconciles on every run ([#1226](https://github.com/mlorentedev/dotfiles/issues/1226)) ([9c44d7a](https://github.com/mlorentedev/dotfiles/commit/9c44d7a7d1d6580baee122868d0807db0206407c))


### Bug Fixes

* **ci:** stop running the jobs a Dependabot PR cannot possibly pass ([#1223](https://github.com/mlorentedev/dotfiles/issues/1223)) ([c9e6674](https://github.com/mlorentedev/dotfiles/commit/c9e66747a1438e290d21b7880540e503f04f4b54))
* **cli-042:** the on-disk credential scan skipped everything under zsh, and missed 40% of secrets ([#1234](https://github.com/mlorentedev/dotfiles/issues/1234)) ([240d28b](https://github.com/mlorentedev/dotfiles/commit/240d28b367b2dd186857f67da1e1f80366143247))
* **doctor:** parse the real ExecStart record, and add one-command post-deploy verification ([#1232](https://github.com/mlorentedev/dotfiles/issues/1232)) ([50203fa](https://github.com/mlorentedev/dotfiles/commit/50203fa221f1c40bee65147d3cf20cb5765f0453))
* **mem:** file session records under the local calendar date and complete their frontmatter ([#1217](https://github.com/mlorentedev/dotfiles/issues/1217)) ([5e18ffa](https://github.com/mlorentedev/dotfiles/commit/5e18ffa82b0448eb36103793e101be564ee2cf08))
* **ssh:** the hub alias still named the instance that was destroyed ([#1211](https://github.com/mlorentedev/dotfiles/issues/1211)) ([b801be0](https://github.com/mlorentedev/dotfiles/commit/b801be009c47582524d9047b04e13d8a2c130615))
* **tests:** make it impossible for a test to launch a GUI application ([#1215](https://github.com/mlorentedev/dotfiles/issues/1215)) ([17d749a](https://github.com/mlorentedev/dotfiles/commit/17d749a8c362231768fa609db930abdc8fb66e7c))

## [0.48.1](https://github.com/mlorentedev/dotfiles/compare/v0.48.0...v0.48.1) (2026-08-23)


### Bug Fixes

* **setup:** derive the harness mirror from the manifest, not a hardcoded list ([#1201](https://github.com/mlorentedev/dotfiles/issues/1201)) ([b04a6c4](https://github.com/mlorentedev/dotfiles/commit/b04a6c4e8c44ccfe7d30e950d74262747fe5cd20))

## [0.48.0](https://github.com/mlorentedev/dotfiles/compare/v0.47.1...v0.48.0) (2026-08-23)


### Features

* **harness:** compile catchup session briefing skill from vault ([#1198](https://github.com/mlorentedev/dotfiles/issues/1198)) ([9c644ec](https://github.com/mlorentedev/dotfiles/commit/9c644eca062ccf55a06ee686b2bdddb6acffdde4))


### Bug Fixes

* **mem:** only project and reference memories archive on age ([#1193](https://github.com/mlorentedev/dotfiles/issues/1193)) ([064f20c](https://github.com/mlorentedev/dotfiles/commit/064f20ce2be2c765f298292042992acb533854b9)), closes [#967](https://github.com/mlorentedev/dotfiles/issues/967)

## [0.47.1](https://github.com/mlorentedev/dotfiles/compare/v0.47.0...v0.47.1) (2026-08-22)


### Bug Fixes

* **setup:** deploy the settings.json env block instead of dropping it ([#1188](https://github.com/mlorentedev/dotfiles/issues/1188)) ([62ea2f3](https://github.com/mlorentedev/dotfiles/commit/62ea2f3df0ee274d59dd947f1d35cc003d20fdae))

## [0.47.0](https://github.com/mlorentedev/dotfiles/compare/v0.46.0...v0.47.0) (2026-08-22)


### Features

* **ai:** integrate Orca ADE overlay and enforce IaC idempotence doctrine ([#1176](https://github.com/mlorentedev/dotfiles/issues/1176)) ([3117fe0](https://github.com/mlorentedev/dotfiles/commit/3117fe013ab1134a5f72fa6637fa482e3901d3ff))
* **ci:** configure CodeRabbit instead of inheriting somebody else's defaults ([#1187](https://github.com/mlorentedev/dotfiles/issues/1187)) ([434bdc4](https://github.com/mlorentedev/dotfiles/commit/434bdc428120f517e04cf0775cd8a9e2359d6631))
* **doctor:** catch a record whose declared tier model-map cannot answer ([#1174](https://github.com/mlorentedev/dotfiles/issues/1174)) ([e54263c](https://github.com/mlorentedev/dotfiles/commit/e54263c390574bbde3fdf5f273b3547ad97c7ac6))
* **harness:** give capabilities the same seam the model tier got ([#1172](https://github.com/mlorentedev/dotfiles/issues/1172)) ([2116a58](https://github.com/mlorentedev/dotfiles/commit/2116a58434eae0b9819c39b509f60cb241c2e7a0))
* **harness:** give model-map's tiers block its first consumer ([#1165](https://github.com/mlorentedev/dotfiles/issues/1165)) ([8e45bd8](https://github.com/mlorentedev/dotfiles/commit/8e45bd8ec7ee45ed6b632d9f21c866d365a1af6a))
* **spec:** refuse to archive on a verdict the reviewer never wrote ([#1178](https://github.com/mlorentedev/dotfiles/issues/1178)) ([81d0b51](https://github.com/mlorentedev/dotfiles/commit/81d0b5193e1f18285bb92d5298749eaa013cb6f1))


### Bug Fixes

* **ci:** a reviewer whose quota is exhausted must not refuse the change ([#1184](https://github.com/mlorentedev/dotfiles/issues/1184)) ([279951f](https://github.com/mlorentedev/dotfiles/commit/279951f9a969a78cd0d0a36701554567c955d5b0))
* **ci:** the attestation check-run must not carry a verdict that will change ([#1185](https://github.com/mlorentedev/dotfiles/issues/1185)) ([4b76def](https://github.com/mlorentedev/dotfiles/commit/4b76defab5f6b41664ebc5f2ef5410a4bbed4ea1))
* **doctor:** key each doctrine marker on its own rule, not another's prose ([#1182](https://github.com/mlorentedev/dotfiles/issues/1182)) ([a91ec35](https://github.com/mlorentedev/dotfiles/commit/a91ec35f238c8252403ef1a2ce368b2b3457953d))
* **harness:** agy takes its model as a launcher flag, so it is an adapter ([#1171](https://github.com/mlorentedev/dotfiles/issues/1171)) ([358e8ca](https://github.com/mlorentedev/dotfiles/commit/358e8caa95fbc7eb8ede88059137d796c7fb8261))
* **harness:** land the capability-map review fixes that [#1172](https://github.com/mlorentedev/dotfiles/issues/1172) merged without ([#1179](https://github.com/mlorentedev/dotfiles/issues/1179)) ([2dd16a2](https://github.com/mlorentedev/dotfiles/commit/2dd16a2c51d4bb5c21e35490da6f44b3269c71be))
* **harness:** make the compact doctrine payload actually compact ([#1181](https://github.com/mlorentedev/dotfiles/issues/1181)) ([b2d3de1](https://github.com/mlorentedev/dotfiles/commit/b2d3de1090b5f43b22c012f0a67c8fcc8b08f185))
* **setup:** jq `// empty` guard silently disabled the whole Claude settings merge ([#1167](https://github.com/mlorentedev/dotfiles/issues/1167)) ([968174d](https://github.com/mlorentedev/dotfiles/commit/968174d5e7b19b3106ca8155c8fe9a645237cec9))
* **spec:** the provider-diverse reviewer arm has never worked ([#1177](https://github.com/mlorentedev/dotfiles/issues/1177)) ([bb9b99b](https://github.com/mlorentedev/dotfiles/commit/bb9b99b89828f081cfde0ef0b6d8f973fe62b641))

## [0.46.0](https://github.com/mlorentedev/dotfiles/compare/v0.45.0...v0.46.0) (2026-08-21)


### Features

* **ai:** optimize multi-agent configurations, documentation, and tooling ([#1089](https://github.com/mlorentedev/dotfiles/issues/1089)) ([3033778](https://github.com/mlorentedev/dotfiles/commit/303377891a9230f09f20b307a873cb38219f3306))
* **harness:** admit mimo-v2.5 to the reviewer pool, with the verdict stated honestly ([#1116](https://github.com/mlorentedev/dotfiles/issues/1116)) ([12d0116](https://github.com/mlorentedev/dotfiles/commit/12d0116bd0291bc020bf4cb0d0b8f5d24b2cace3))
* **harness:** bind pr triage queue to DoD evidence gate and handoff skill ([#1131](https://github.com/mlorentedev/dotfiles/issues/1131)) ([33d797e](https://github.com/mlorentedev/dotfiles/commit/33d797ef32c43beff80edddf5afa742d99376337))
* **harness:** build model-map.json, its schema, and the first doctor check over a registry ([#1143](https://github.com/mlorentedev/dotfiles/issues/1143)) ([e22a4d0](https://github.com/mlorentedev/dotfiles/commit/e22a4d0b18f3d367f4821ae9c39cdf534409882d))
* **harness:** forbid printing a secret, in the doctrine every agent receives ([#1114](https://github.com/mlorentedev/dotfiles/issues/1114)) ([fcaa403](https://github.com/mlorentedev/dotfiles/commit/fcaa403c2faeb3f45ca3db3722ac7a94c7e817e9))
* one-liner curl bootstrap install.sh (IDEAS-005) ([#1108](https://github.com/mlorentedev/dotfiles/issues/1108)) ([ef66667](https://github.com/mlorentedev/dotfiles/commit/ef66667ce217e50dfdccc6c8bb2f3e58970ea636))
* **spec:** scaffold features.json during dotf spec init ([#1127](https://github.com/mlorentedev/dotfiles/issues/1127)) ([6e51d65](https://github.com/mlorentedev/dotfiles/commit/6e51d659a01737b0a5514a80df381d93ac00a1bc)), closes [#1076](https://github.com/mlorentedev/dotfiles/issues/1076)


### Bug Fixes

* **attestation:** recognize CodeRabbit clean-review comment as attestation ([#1125](https://github.com/mlorentedev/dotfiles/issues/1125)) ([9fa89f9](https://github.com/mlorentedev/dotfiles/commit/9fa89f9cf6bf99b7e753b394bd5ab17e8a152307)), closes [#1122](https://github.com/mlorentedev/dotfiles/issues/1122)
* **ci:** cap the reviewer's inference demand, and queue instead of failing ([#1110](https://github.com/mlorentedev/dotfiles/issues/1110)) ([fc9d6f0](https://github.com/mlorentedev/dotfiles/commit/fc9d6f06d13f353a0d75d2115e28e9f2ef271f1d))
* **ci:** fail the review job when it published no review ([#1109](https://github.com/mlorentedev/dotfiles/issues/1109)) ([237b595](https://github.com/mlorentedev/dotfiles/commit/237b595fe7fc8b3eed50406785210e9743a092f0))
* **ci:** filter pr-agent issue_comment trigger to slash commands only ([#1135](https://github.com/mlorentedev/dotfiles/issues/1135)) ([881546f](https://github.com/mlorentedev/dotfiles/commit/881546f2feef718a1f6126414a79f6d9d20a408b)), closes [#1134](https://github.com/mlorentedev/dotfiles/issues/1134)
* **ci:** gate release PRs out of the reviewer where the tool cannot ignore it ([#1102](https://github.com/mlorentedev/dotfiles/issues/1102)) ([26c1bcf](https://github.com/mlorentedev/dotfiles/commit/26c1bcf7e5f3587367181c8adc119ec2155dbe3a))
* **ci:** give PR-Agent its own model lane so it stops starving the spec-review gate ([#1150](https://github.com/mlorentedev/dotfiles/issues/1150)) ([c368c03](https://github.com/mlorentedev/dotfiles/commit/c368c0325cbaf56c57fbda7a9be5fe8e61860257)), closes [#1149](https://github.com/mlorentedev/dotfiles/issues/1149)
* **ci:** remove job-level concurrency lock in pr-agent workflow ([#1146](https://github.com/mlorentedev/dotfiles/issues/1146)) ([e09de16](https://github.com/mlorentedev/dotfiles/commit/e09de165963cdae32e9d8c9f4e440553f0d78b7e))
* **ci:** trigger review-attestation on pull_request_review events ([#1121](https://github.com/mlorentedev/dotfiles/issues/1121)) ([2261a8e](https://github.com/mlorentedev/dotfiles/commit/2261a8e9db4dae176e73de8ec776abec81fa480e)), closes [#1115](https://github.com/mlorentedev/dotfiles/issues/1115)
* **docs:** auto-discover instruction files and wire check-doc-paths into CI and pre-commit ([#1133](https://github.com/mlorentedev/dotfiles/issues/1133)) ([5b39065](https://github.com/mlorentedev/dotfiles/commit/5b390653d3d950df111d38c7d9390829b8f99b0a))
* **doctor:** report stale or unsynced bw cache as WARN in mapping check ([#1144](https://github.com/mlorentedev/dotfiles/issues/1144)) ([10868de](https://github.com/mlorentedev/dotfiles/commit/10868debc96d3fce86187e802c9282d962292a90)), closes [#1015](https://github.com/mlorentedev/dotfiles/issues/1015)
* **doctor:** support file-authority backend, auto-tune orca hook, and bump pi pin ([#1082](https://github.com/mlorentedev/dotfiles/issues/1082)) ([31bcc7f](https://github.com/mlorentedev/dotfiles/commit/31bcc7fc9ba84c260cb5247d79d3a1ae589abe2b))
* **env:** prefer cwd worktree over DOTFILES_REPO_DIR in RepoDir ([#1129](https://github.com/mlorentedev/dotfiles/issues/1129)) ([2340479](https://github.com/mlorentedev/dotfiles/commit/2340479abd7054fda44e12e8ab48b1b654317cc3)), closes [#939](https://github.com/mlorentedev/dotfiles/issues/939)
* **harness:** land the model-map review fixes that [#1143](https://github.com/mlorentedev/dotfiles/issues/1143) merged without ([#1155](https://github.com/mlorentedev/dotfiles/issues/1155)) ([63acd91](https://github.com/mlorentedev/dotfiles/commit/63acd91f7f299255dd1a825933f3c32322dcd585)), closes [#1124](https://github.com/mlorentedev/dotfiles/issues/1124)
* **hooks:** stop the global dispatcher re-entering pre-commit's own store ([#1097](https://github.com/mlorentedev/dotfiles/issues/1097)) ([fad8b12](https://github.com/mlorentedev/dotfiles/commit/fad8b125974a99e51b4d2f4f0de86528a9583aea))
* **prtriage:** address 5 Minor findings from mimo-v2.5 review ([#1123](https://github.com/mlorentedev/dotfiles/issues/1123)) ([6d6e1ad](https://github.com/mlorentedev/dotfiles/commit/6d6e1ad29775d900a3b3ffb53a70cb8206171c83))
* **review:** close the triage loop — the marker gets a writer, and the queue gets a wake-up ([#1101](https://github.com/mlorentedev/dotfiles/issues/1101)) ([44c9417](https://github.com/mlorentedev/dotfiles/commit/44c94173297b250a275ce6f3284c498f3c00fac2))
* **secrets:** enforce AGE_VERSION across setup, integration container, and doctor ([#1120](https://github.com/mlorentedev/dotfiles/issues/1120)) ([429d4ca](https://github.com/mlorentedev/dotfiles/commit/429d4cadc9127f1f1da3c16337a78f7da3c4834d))
* **secrets:** rewrite non-existent --split flag in migrate messages ([#1130](https://github.com/mlorentedev/dotfiles/issues/1130)) ([ae79bbf](https://github.com/mlorentedev/dotfiles/commit/ae79bbfdef034ef832c0bd1c905958cc3d9166e7)), closes [#941](https://github.com/mlorentedev/dotfiles/issues/941)
* **spec:** compare content instead of ancestry for review staleness ([#1126](https://github.com/mlorentedev/dotfiles/issues/1126)) ([380fe75](https://github.com/mlorentedev/dotfiles/commit/380fe7509938da01686d13779c8c8839e027ad88))
* **spec:** scope secret injection during review launch via pool secret_id ([#1132](https://github.com/mlorentedev/dotfiles/issues/1132)) ([40045ea](https://github.com/mlorentedev/dotfiles/commit/40045ea7db92c7135d9391c390f7859985456112)), closes [#1025](https://github.com/mlorentedev/dotfiles/issues/1025)

## [0.45.0](https://github.com/mlorentedev/dotfiles/compare/v0.44.0...v0.45.0) (2026-08-19)


### Features

* **cli:** add dotf search and dotf harness suggest commands ([#1067](https://github.com/mlorentedev/dotfiles/issues/1067)) ([27c7ccf](https://github.com/mlorentedev/dotfiles/commit/27c7ccf41edcdca2a6839d94712a6ea6482dd2ff))
* **harness:** skill dependencies resolution and full trigger catalog ([#1070](https://github.com/mlorentedev/dotfiles/issues/1070)) ([6c17f9c](https://github.com/mlorentedev/dotfiles/commit/6c17f9ce5832ef239a59604af46494397db9fa3b))
* **secrets:** check the age root has not drifted, and whether the escrow still describes the vault ([#1077](https://github.com/mlorentedev/dotfiles/issues/1077)) ([#1079](https://github.com/mlorentedev/dotfiles/issues/1079)) ([c805e8f](https://github.com/mlorentedev/dotfiles/commit/c805e8f4b2ee0c0e8c758b5018a176d206417fd5))
* **secrets:** give the age root a backend, so the inventory contains its own root ([#937](https://github.com/mlorentedev/dotfiles/issues/937)) ([#1075](https://github.com/mlorentedev/dotfiles/issues/1075)) ([10204ae](https://github.com/mlorentedev/dotfiles/commit/10204ae2bac6c4ba65d9919d51b5671c9bf7ee87))


### Bug Fixes

* **ci:** a review attests only from a member or a declared reviewer ([#1033](https://github.com/mlorentedev/dotfiles/issues/1033)) ([#1071](https://github.com/mlorentedev/dotfiles/issues/1071)) ([2bac1c5](https://github.com/mlorentedev/dotfiles/commit/2bac1c59e8f2bdabe79eac8c4ee3a077bfd527b2))
* **ci:** pin the reviewer action by commit and gate its comment trigger on membership ([#1078](https://github.com/mlorentedev/dotfiles/issues/1078)) ([6e1c114](https://github.com/mlorentedev/dotfiles/commit/6e1c114f0240a9fcbedc037f835a4545b154a9b8))
* **harness:** filter neutral metadata on skill deploy for unconditional discovery ([#1080](https://github.com/mlorentedev/dotfiles/issues/1080)) ([#1081](https://github.com/mlorentedev/dotfiles/issues/1081)) ([ecceec8](https://github.com/mlorentedev/dotfiles/commit/ecceec83f32ffcae19f2b5b7f1a7f3aebdbabe26))

## [0.44.0](https://github.com/mlorentedev/dotfiles/compare/v0.43.0...v0.44.0) (2026-08-18)


### Features

* **ci:** make the reviewer check harness compliance by default ([#786](https://github.com/mlorentedev/dotfiles/issues/786)) ([#1044](https://github.com/mlorentedev/dotfiles/issues/1044)) ([a863730](https://github.com/mlorentedev/dotfiles/commit/a863730cf70c365b982a6b60f0155da55ebdcb27))
* **ci:** review every push, because the doctrine already says we do ([#786](https://github.com/mlorentedev/dotfiles/issues/786)) ([#1058](https://github.com/mlorentedev/dotfiles/issues/1058)) ([bb2fa23](https://github.com/mlorentedev/dotfiles/commit/bb2fa2383a2990c741a1a0148790f4973f40da8f))
* **ci:** stop reviewing release PRs, and stop demanding a review of them ([#786](https://github.com/mlorentedev/dotfiles/issues/786)) ([#1065](https://github.com/mlorentedev/dotfiles/issues/1065)) ([a889bd6](https://github.com/mlorentedev/dotfiles/commit/a889bd6c2c3831bc0ee286c703d52ae15951b782))
* **cli:** dotf pr triage-queue — the wake-up the review loop never had ([#1052](https://github.com/mlorentedev/dotfiles/issues/1052)) ([#1057](https://github.com/mlorentedev/dotfiles/issues/1057)) ([1a29340](https://github.com/mlorentedev/dotfiles/commit/1a29340f8696d0aa51a51752a3dd0bd29d7ab37a))
* **harness:** reconcile spec subcommands, add deployed doctrine probes, and enhance router ([#1046](https://github.com/mlorentedev/dotfiles/issues/1046)) ([ab4f303](https://github.com/mlorentedev/dotfiles/commit/ab4f303eb88d1800ec1dfd35979582776ed76ebf))


### Bug Fixes

* **ci:** align the pr-agent trigger list with PR-Agent's own event gate ([#1054](https://github.com/mlorentedev/dotfiles/issues/1054)) ([e608989](https://github.com/mlorentedev/dotfiles/commit/e608989e5437f6e465279c83ee09aa34b9aba8e5)), closes [#1053](https://github.com/mlorentedev/dotfiles/issues/1053)
* **ci:** install age from the pinned release on Linux, and verify what it got ([#1059](https://github.com/mlorentedev/dotfiles/issues/1059)) ([e63a35b](https://github.com/mlorentedev/dotfiles/commit/e63a35b5c84aa75eb86ac25ee221da3aebf0f5df))
* **ci:** let a declared reviewer attest with comment-shaped output ([#1047](https://github.com/mlorentedev/dotfiles/issues/1047)) ([7d33378](https://github.com/mlorentedev/dotfiles/commit/7d33378fed6593669095d8aead28f9b43bea085a))
* **ci:** re-evaluate the review gate when our own reviewer finishes ([#1052](https://github.com/mlorentedev/dotfiles/issues/1052), [#1041](https://github.com/mlorentedev/dotfiles/issues/1041)) ([#1056](https://github.com/mlorentedev/dotfiles/issues/1056)) ([1a5f3cf](https://github.com/mlorentedev/dotfiles/commit/1a5f3cf4cf77805977066b337f94f3ee2176a366))
* **ci:** stop PR-Agent cancelling its own review when a bot comments ([#1042](https://github.com/mlorentedev/dotfiles/issues/1042)) ([1cf1f56](https://github.com/mlorentedev/dotfiles/commit/1cf1f56f69eb53e6649be341205a30648eeaf286))

## [0.43.0](https://github.com/mlorentedev/dotfiles/compare/v0.42.0...v0.43.0) (2026-08-16)


### Features

* **ci:** add PR-Agent on NaN inference, so review capacity stops gating throughput ([#1032](https://github.com/mlorentedev/dotfiles/issues/1032)) ([23c5716](https://github.com/mlorentedev/dotfiles/commit/23c57169af3c8ae019321f3f1e88f71e4e537ee2))
* **ci:** make a green check mean reviewed, not merely un-failed ([#1019](https://github.com/mlorentedev/dotfiles/issues/1019)) ([e033302](https://github.com/mlorentedev/dotfiles/commit/e033302489a9e446efa8e38253cb7aa5a7b8a590))
* **cli:** add `dotf deploy`, one implementation of agent-config deployment ([#1027](https://github.com/mlorentedev/dotfiles/issues/1027)) ([bf7d33e](https://github.com/mlorentedev/dotfiles/commit/bf7d33e5d9f132cf556bb0cea6d083b898c5890f)), closes [#1023](https://github.com/mlorentedev/dotfiles/issues/1023)


### Bug Fixes

* **ci:** make the failing attestation step say why, not just exit 1 ([#906](https://github.com/mlorentedev/dotfiles/issues/906)) ([#1029](https://github.com/mlorentedev/dotfiles/issues/1029)) ([a724086](https://github.com/mlorentedev/dotfiles/commit/a724086b6e747c7119cb59b49ed2d2b914f11932))
* **pi:** resolve the API key at runtime, so no config carries a credential ([#1026](https://github.com/mlorentedev/dotfiles/issues/1026)) ([db5b314](https://github.com/mlorentedev/dotfiles/commit/db5b314d1b2f33ddf681b29c0211fe52db77decb)), closes [#987](https://github.com/mlorentedev/dotfiles/issues/987)
* **spec:** stop refusing a passing review over punctuation ([#963](https://github.com/mlorentedev/dotfiles/issues/963)) ([#1031](https://github.com/mlorentedev/dotfiles/issues/1031)) ([2f7bfd5](https://github.com/mlorentedev/dotfiles/commit/2f7bfd5b61db9fd1fc90c6b5b541a1954e7414a5))

## [0.42.0](https://github.com/mlorentedev/dotfiles/compare/v0.41.0...v0.42.0) (2026-08-16)


### Features

* **secrets:** add `dotf secrets probe`, an instrument that cannot print a credential ([#1022](https://github.com/mlorentedev/dotfiles/issues/1022)) ([a2a5760](https://github.com/mlorentedev/dotfiles/commit/a2a57605a9f862fbf17c4a908d9ee10dd413193c)), closes [#1012](https://github.com/mlorentedev/dotfiles/issues/1012)
* **secrets:** put the DR escrow on the USB, and write down what the backup policy is ([#1000](https://github.com/mlorentedev/dotfiles/issues/1000)) ([#1017](https://github.com/mlorentedev/dotfiles/issues/1017)) ([a8d7eb2](https://github.com/mlorentedev/dotfiles/commit/a8d7eb28c8ab5ce9ef022320c5e4703c3f08ed06))


### Bug Fixes

* **doctor:** key DR escrow severity to real exposure, not a flat policy ([#1006](https://github.com/mlorentedev/dotfiles/issues/1006)) ([8053396](https://github.com/mlorentedev/dotfiles/commit/8053396b0c7c7c6c09eb2e48abd51d7200d12cc7)), closes [#997](https://github.com/mlorentedev/dotfiles/issues/997)
* **secrets:** give the write path the bw serve seam the read path already had ([#1007](https://github.com/mlorentedev/dotfiles/issues/1007)) ([fe2f191](https://github.com/mlorentedev/dotfiles/commit/fe2f19136dc91e90dc2bab88bf8c7f04467fc7c1)), closes [#993](https://github.com/mlorentedev/dotfiles/issues/993)
* **secrets:** let verify report a broken registry instead of dying on it ([#1020](https://github.com/mlorentedev/dotfiles/issues/1020)) ([718c895](https://github.com/mlorentedev/dotfiles/commit/718c8958b7626763b8999d1d1a3e8f467098fc67)), closes [#1004](https://github.com/mlorentedev/dotfiles/issues/1004)
* **secrets:** stop probing /status, which poisons the daemon's item reads ([#1018](https://github.com/mlorentedev/dotfiles/issues/1018)) ([e66120f](https://github.com/mlorentedev/dotfiles/commit/e66120f17e34084c02e3a44c9c7b4ce73c575e1a)), closes [#988](https://github.com/mlorentedev/dotfiles/issues/988)
* **spec:** stop the archive gate from refusing its own review's output ([#1009](https://github.com/mlorentedev/dotfiles/issues/1009)) ([bb3b75d](https://github.com/mlorentedev/dotfiles/commit/bb3b75dc37e746f664849bb9acad12434c7cdf10))

## [0.41.0](https://github.com/mlorentedev/dotfiles/compare/v0.40.0...v0.41.0) (2026-08-15)


### Features

* **harness:** a PR you open is watched, not abandoned (HARNESS-072-pr-stewardship, [#963](https://github.com/mlorentedev/dotfiles/issues/963)) ([#986](https://github.com/mlorentedev/dotfiles/issues/986)) ([62d2e84](https://github.com/mlorentedev/dotfiles/commit/62d2e84efefbfcdb5fb36bd57f77479a9111dffe))
* **secrets:** add `dotf secrets rotate` — replace a credential and prove the replacement took ([#1003](https://github.com/mlorentedev/dotfiles/issues/1003)) ([3644847](https://github.com/mlorentedev/dotfiles/commit/36448473dde9b04dc8c8aba439aeedb1ec1b0fab)), closes [#988](https://github.com/mlorentedev/dotfiles/issues/988)


### Bug Fixes

* **doctor:** repair the PAT-expiry check and the tagged-union consumers behind it ([#984](https://github.com/mlorentedev/dotfiles/issues/984)) ([6252eba](https://github.com/mlorentedev/dotfiles/commit/6252eba356d8c4c5341e85d24cc4560eefc7417a)), closes [#972](https://github.com/mlorentedev/dotfiles/issues/972)
* **secrets:** map DOCKERHUB_TOKEN to the scoped PAT, and detect registry/vault drift ([#990](https://github.com/mlorentedev/dotfiles/issues/990)) ([4e559a7](https://github.com/mlorentedev/dotfiles/commit/4e559a7fd592129c98da6822f177b6f05d2b1c8f)), closes [#985](https://github.com/mlorentedev/dotfiles/issues/985)
* **spec:** stop announcing a review that never started ([#994](https://github.com/mlorentedev/dotfiles/issues/994)) ([6991425](https://github.com/mlorentedev/dotfiles/commit/69914254d31996e58812e6684aee6a92a568dee5)), closes [#989](https://github.com/mlorentedev/dotfiles/issues/989)
* **spec:** store the review transcript's events, not its streaming frames ([#999](https://github.com/mlorentedev/dotfiles/issues/999)) ([77d7b7e](https://github.com/mlorentedev/dotfiles/commit/77d7b7e44b0bd9c739ed2836d4d5ab3db866f652)), closes [#995](https://github.com/mlorentedev/dotfiles/issues/995)

## [0.40.0](https://github.com/mlorentedev/dotfiles/compare/v0.39.0...v0.40.0) (2026-08-15)


### Features

* **harness:** add file-path pattern trigger resolution to dotf ([#981](https://github.com/mlorentedev/dotfiles/issues/981)) ([fc1af0c](https://github.com/mlorentedev/dotfiles/commit/fc1af0ce45bfe4d42f57395e8f314cebe5387104))
* **secrets:** bw serve read-path backend (dotf secrets unlock, no ambient BW_SESSION) ([#975](https://github.com/mlorentedev/dotfiles/issues/975)) ([9ee5860](https://github.com/mlorentedev/dotfiles/commit/9ee58605cf5ad048ad65f03c9f6a7908784df147))
* **spec:** close HARNESS-071's AC7 and archive it — the reviewer pool gate, reviewed by the pool ([#978](https://github.com/mlorentedev/dotfiles/issues/978)) ([a1bf0f7](https://github.com/mlorentedev/dotfiles/commit/a1bf0f78ef4756d0ccf70f538f13cc3ed11c178f)), closes [#955](https://github.com/mlorentedev/dotfiles/issues/955)


### Bug Fixes

* **doctor:** dispatch the secrets-integrity check on backend, not on File (BUG-077, [#969](https://github.com/mlorentedev/dotfiles/issues/969)) ([#973](https://github.com/mlorentedev/dotfiles/issues/973)) ([271e4ec](https://github.com/mlorentedev/dotfiles/commit/271e4ec9cb9fac0a426524e1e4f53df7427aeb0c))
* **secrets:** parse bw serve's /status template-wrapped shape ([#979](https://github.com/mlorentedev/dotfiles/issues/979)) ([fa21694](https://github.com/mlorentedev/dotfiles/commit/fa2169471ac9069a7400fd140568af3c1af8bf6e))
* **shell:** scope the agent secret wrappers with --only, and stop wrapping agy ([#977](https://github.com/mlorentedev/dotfiles/issues/977)) ([0f01a1e](https://github.com/mlorentedev/dotfiles/commit/0f01a1e56794f232f6d5d545f0f96bb5637a2dfd)), closes [#976](https://github.com/mlorentedev/dotfiles/issues/976)

## [0.39.0](https://github.com/mlorentedev/dotfiles/compare/v0.38.0...v0.39.0) (2026-08-14)


### Features

* **secrets:** give the registry a Bitwarden folder taxonomy (OPS-028) ([#957](https://github.com/mlorentedev/dotfiles/issues/957)) ([092cb80](https://github.com/mlorentedev/dotfiles/commit/092cb80de77e142967543733ddb0454cb7c51360))
* **secrets:** migrate 22 dev secrets from the age store to Bitwarden ([#585](https://github.com/mlorentedev/dotfiles/issues/585)) ([#961](https://github.com/mlorentedev/dotfiles/issues/961)) ([a42cc67](https://github.com/mlorentedev/dotfiles/commit/a42cc67f11e56bb28da9ac6145d2a6db4ad3a62b))
* **secrets:** migrate 5 file secrets from the age store to Bitwarden (CLI-024-secrets-file-migrate, [#964](https://github.com/mlorentedev/dotfiles/issues/964)) ([#965](https://github.com/mlorentedev/dotfiles/issues/965)) ([852bbaa](https://github.com/mlorentedev/dotfiles/commit/852bbaaeb76e5d5759b889f006ba179a40ebffeb))
* **spec:** dotf spec review — launch the pooled reviewer with an explicit pin, watchably ([#959](https://github.com/mlorentedev/dotfiles/issues/959)) ([a6a1458](https://github.com/mlorentedev/dotfiles/commit/a6a1458db650656366d98f11d175c952fe0fa636)), closes [#955](https://github.com/mlorentedev/dotfiles/issues/955)
* **spec:** enforce adversarial-review independence with a reviewer pool gate ([#958](https://github.com/mlorentedev/dotfiles/issues/958)) ([2f222dd](https://github.com/mlorentedev/dotfiles/commit/2f222ddc439c3043d78d9471904c80aa0d7f22cf)), closes [#955](https://github.com/mlorentedev/dotfiles/issues/955)


### Bug Fixes

* **doctor:** prove the live secrets SSOT by reach, not by PATH presence (BUG-074) ([#950](https://github.com/mlorentedev/dotfiles/issues/950)) ([9317a8d](https://github.com/mlorentedev/dotfiles/commit/9317a8d5c2266f8d80aaace15c175c0997d6f9b1))
* **harness:** converge the deploy engine — prune, drift detection, all six surfaces ([#948](https://github.com/mlorentedev/dotfiles/issues/948)) ([18ccd60](https://github.com/mlorentedev/dotfiles/commit/18ccd60fb22f8bc9d7f41e1d630feca0a5a469cf))
* **review:** land CodeRabbit findings from PR [#948](https://github.com/mlorentedev/dotfiles/issues/948) that missed the merge ([#954](https://github.com/mlorentedev/dotfiles/issues/954)) ([fcc5601](https://github.com/mlorentedev/dotfiles/commit/fcc56013b398d7fe7ca0a5a0625384ac64a6726d))
* **spec:** land the agy reviewer fixes that missed [#959](https://github.com/mlorentedev/dotfiles/issues/959)'s merge ([#966](https://github.com/mlorentedev/dotfiles/issues/966)) ([9b2b399](https://github.com/mlorentedev/dotfiles/commit/9b2b3991c4f949610e56e660953d101278e78012)), closes [#955](https://github.com/mlorentedev/dotfiles/issues/955)
* **tests:** repair four guards that pass without checking what they claim ([#949](https://github.com/mlorentedev/dotfiles/issues/949)) ([e5a2ecc](https://github.com/mlorentedev/dotfiles/commit/e5a2ecc7c2890342e01967da578d5945d58a1f2e))

## [0.38.0](https://github.com/mlorentedev/dotfiles/compare/v0.37.1...v0.38.0) (2026-08-12)


### Features

* **harness:** stamp committed skill/agent records with provenance (HARNESS-069) ([#927](https://github.com/mlorentedev/dotfiles/issues/927)) ([147e4ed](https://github.com/mlorentedev/dotfiles/commit/147e4ed23e42f4865940630ce3076fe7f175c1d4))


### Bug Fixes

* **crystallize:** add log_error to the standalone fallback (BUG-065) ([#932](https://github.com/mlorentedev/dotfiles/issues/932)) ([d090032](https://github.com/mlorentedev/dotfiles/commit/d0900321c7d860cb2effe025ca434b00b9aa8832))
* **docs:** govern nested READMEs and archive DOCS-013 through the review gate ([#926](https://github.com/mlorentedev/dotfiles/issues/926)) ([2eb1a5d](https://github.com/mlorentedev/dotfiles/commit/2eb1a5df0133b7773c9068311b223fc8c9cf7d65))
* **doctor:** detect a vault linked worktree as a checkout, not absent (BUG-053, [#806](https://github.com/mlorentedev/dotfiles/issues/806)) ([#931](https://github.com/mlorentedev/dotfiles/issues/931)) ([5116466](https://github.com/mlorentedev/dotfiles/commit/5116466e7d19da89d2e8104030c89d5523861e8b))
* **harness:** PowerShell twin never stripped a record's own generated_* fields (HARNESS-069) ([#934](https://github.com/mlorentedev/dotfiles/issues/934)) ([8170e96](https://github.com/mlorentedev/dotfiles/commit/8170e96cfbffedf1f8067750a4cb47ae309278cb))
* **tests:** widen the git-alias collision guard past its 4-char cap (BUG-045) ([#935](https://github.com/mlorentedev/dotfiles/issues/935)) ([ac826ed](https://github.com/mlorentedev/dotfiles/commit/ac826edc7dcfdc8afb9329ae89055347575c9616))
* **vault:** drop the redundant --vault flag from 4 of 5 obsidian_cmd callers ([#891](https://github.com/mlorentedev/dotfiles/issues/891)) ([#930](https://github.com/mlorentedev/dotfiles/issues/930)) ([cde3bcc](https://github.com/mlorentedev/dotfiles/commit/cde3bcc573679aee4e6bc2c757f95127ed578f60))

## [0.37.1](https://github.com/mlorentedev/dotfiles/compare/v0.37.0...v0.37.1) (2026-08-11)


### Bug Fixes

* **ci:** pin golangci-lint from versions.conf instead of the action default ([#920](https://github.com/mlorentedev/dotfiles/issues/920)) ([2c4b506](https://github.com/mlorentedev/dotfiles/commit/2c4b506d754d2fc4bc4fe9d738dfbd293fb6bd54))
* **docs:** apply three rounds of adversarial review to the doc-path guard (DOCS-013) ([#924](https://github.com/mlorentedev/dotfiles/issues/924)) ([caa7af5](https://github.com/mlorentedev/dotfiles/commit/caa7af559dc28c0cb97d31f61a413c705dd914b2))
* **spec:** accept digit-bearing AREAs in feature-ids and reconcile the seven copies ([#923](https://github.com/mlorentedev/dotfiles/issues/923)) ([6267bea](https://github.com/mlorentedev/dotfiles/commit/6267bea16ba11039f8c39e488e93c6b40e845c6c))

## [0.37.0](https://github.com/mlorentedev/dotfiles/compare/v0.36.0...v0.37.0) (2026-08-10)


### Features

* **agents:** propose the adversarial review in the verification window ([#896](https://github.com/mlorentedev/dotfiles/issues/896)) ([db272ac](https://github.com/mlorentedev/dotfiles/commit/db272ac4d8a97154f609f48c913d7b443c2c1e7a))


### Bug Fixes

* **doctor:** stop dotf doctor failing on a healthy Windows box (BUG-052) ([#910](https://github.com/mlorentedev/dotfiles/issues/910)) ([eff2ece](https://github.com/mlorentedev/dotfiles/commit/eff2ece4376d07d36454e7054a3017b26cf17f22))
* **git-hooks:** force LF eol on tracked hooks so Windows commits run ([#913](https://github.com/mlorentedev/dotfiles/issues/913)) ([b8d3897](https://github.com/mlorentedev/dotfiles/commit/b8d38975eb740c0079d5c216d596bb6921bae699)), closes [#911](https://github.com/mlorentedev/dotfiles/issues/911)

## [0.36.0](https://github.com/mlorentedev/dotfiles/compare/v0.35.1...v0.36.0) (2026-08-10)


### Features

* **vault:** dotf vault crystallize, byte-identical to the shell oracle (CLI-021 increment 1) ([#882](https://github.com/mlorentedev/dotfiles/issues/882)) ([a697b54](https://github.com/mlorentedev/dotfiles/commit/a697b54cac0d652fd013ebc3b44e867b22ea7e96))


### Bug Fixes

* **bitacora:** back-fill the board via GraphQL, not gh project item-add ([#888](https://github.com/mlorentedev/dotfiles/issues/888)) ([aef6786](https://github.com/mlorentedev/dotfiles/commit/aef6786085a2a86b93575f30887021fa9f4f81bb))
* **ci:** read spec-gate PR metadata live instead of from the event payload ([#885](https://github.com/mlorentedev/dotfiles/issues/885)) ([327feac](https://github.com/mlorentedev/dotfiles/commit/327feac2ef99de7f4a8ac039fd20304012750294))

## [0.35.1](https://github.com/mlorentedev/dotfiles/compare/v0.35.0...v0.35.1) (2026-08-09)


### Bug Fixes

* **ci:** harden the reconciler's own reporting path against the same -e trap ([#872](https://github.com/mlorentedev/dotfiles/issues/872)) ([58f7419](https://github.com/mlorentedev/dotfiles/commit/58f74195d1f687ddc39b40f8eefcc75adcb94fee))
* **ci:** make the bitacora reconciler's error handling reachable under Actions' injected -e ([#870](https://github.com/mlorentedev/dotfiles/issues/870)) ([2752873](https://github.com/mlorentedev/dotfiles/commit/27528733f274509e0933b0b452e7f5391aaba0ee))

## [0.35.0](https://github.com/mlorentedev/dotfiles/compare/v0.34.0...v0.35.0) (2026-08-09)


### Features

* **doctor:** migrate YAML-wrapped MEMORY.md files back to plain markdown ([#866](https://github.com/mlorentedev/dotfiles/issues/866)) ([13a8b6d](https://github.com/mlorentedev/dotfiles/commit/13a8b6d97cbb8c8fb3c5ee794b4dd06e47aafe88)), closes [#864](https://github.com/mlorentedev/dotfiles/issues/864)


### Bug Fixes

* **crystallize:** refuse a YAML-wrapped MEMORY.md instead of corrupting it ([#862](https://github.com/mlorentedev/dotfiles/issues/862)) ([9caedc1](https://github.com/mlorentedev/dotfiles/commit/9caedc12b7315438eea3994ef20ce1e8d932af15)), closes [#857](https://github.com/mlorentedev/dotfiles/issues/857)

## [0.34.0](https://github.com/mlorentedev/dotfiles/compare/v0.33.1...v0.34.0) (2026-08-08)


### Features

* **doctor:** probe whether guards fire, not where their files sit ([#853](https://github.com/mlorentedev/dotfiles/issues/853)) ([b412597](https://github.com/mlorentedev/dotfiles/commit/b412597167d170d5d89c6119220aa2cd17f8b5a8))


### Bug Fixes

* **hooks:** pin default_stages so a stage-agnostic hook runs once, not once per hook type ([#846](https://github.com/mlorentedev/dotfiles/issues/846)) ([256808a](https://github.com/mlorentedev/dotfiles/commit/256808a0d985ceefb94346238b024baa75b1d05f))
* keep the Session Handoff block last when crystallizing MEMORY.md ([#851](https://github.com/mlorentedev/dotfiles/issues/851)) ([dbe91db](https://github.com/mlorentedev/dotfiles/commit/dbe91db65d22182e94f7b1df6ed6a8842f77dd6b))

## [0.33.1](https://github.com/mlorentedev/dotfiles/compare/v0.33.0...v0.33.1) (2026-08-08)


### Bug Fixes

* **hooks:** pass --hook-dir so the dispatcher fallback does not abort every commit ([#840](https://github.com/mlorentedev/dotfiles/issues/840)) ([c938d1e](https://github.com/mlorentedev/dotfiles/commit/c938d1efbe8b6039baabe8ea7dae0a3fba569dee)), closes [#837](https://github.com/mlorentedev/dotfiles/issues/837)

## [0.33.0](https://github.com/mlorentedev/dotfiles/compare/v0.32.2...v0.33.0) (2026-08-08)


### Features

* **harness:** bind the standing orders to the moment a change is declared done ([#821](https://github.com/mlorentedev/dotfiles/issues/821)) ([5d2a477](https://github.com/mlorentedev/dotfiles/commit/5d2a4778a25502ab00118c06a0b8358044c9acfb)), closes [#820](https://github.com/mlorentedev/dotfiles/issues/820)
* **harness:** deliver doctrine to agy and codex, sized to what each platform reads ([#819](https://github.com/mlorentedev/dotfiles/issues/819)) ([b4ab91e](https://github.com/mlorentedev/dotfiles/commit/b4ab91e2c0a46779dea0ed3e30590c46e49d12fe))
* **harness:** enforce the PR sizing policy on the compact-doctrine agents ([#830](https://github.com/mlorentedev/dotfiles/issues/830)) ([48fff1a](https://github.com/mlorentedev/dotfiles/commit/48fff1a19c2b73ac2f4b61e5e33bada973283114))
* **harness:** one frontmatter contract for the skill library, enforced by the engine ([#826](https://github.com/mlorentedev/dotfiles/issues/826)) ([256f597](https://github.com/mlorentedev/dotfiles/commit/256f5979dc0256f01ee2c65016f74a329adbfac9)), closes [#823](https://github.com/mlorentedev/dotfiles/issues/823)
* **skills:** add pr-review-triage, the disposition step after a PR comes back ([#822](https://github.com/mlorentedev/dotfiles/issues/822)) ([36dd38a](https://github.com/mlorentedev/dotfiles/commit/36dd38ab9e72926ad1b39f4f80cc291074479860))
* **tmux:** add a ~/.tmux.conf.local override seam ([#788](https://github.com/mlorentedev/dotfiles/issues/788)) ([326a020](https://github.com/mlorentedev/dotfiles/commit/326a020556c762f73461da67c55b8a5f4cb4f98b))


### Bug Fixes

* **bitacora:** stop the board losing items on an API failure ([#813](https://github.com/mlorentedev/dotfiles/issues/813)) ([cb9d070](https://github.com/mlorentedev/dotfiles/commit/cb9d0700088881084bc95215757a0e768c03ec0f)), closes [#809](https://github.com/mlorentedev/dotfiles/issues/809)
* **guard:** test hooksPath effectiveness, not string equality ([#801](https://github.com/mlorentedev/dotfiles/issues/801)) ([804ee32](https://github.com/mlorentedev/dotfiles/commit/804ee32b8ef3a2056be560aca37278f31327b14d)), closes [#766](https://github.com/mlorentedev/dotfiles/issues/766)
* **harness:** report unmanaged skill copies at deploy, and unfence five portable skills ([#812](https://github.com/mlorentedev/dotfiles/issues/812)) ([ee146be](https://github.com/mlorentedev/dotfiles/commit/ee146bee33082253a9afea6fcde295059c7bd5d3))
* **hive-upgrade:** distinguish a missing install from an idle no-op (AI-028 PR1) ([#796](https://github.com/mlorentedev/dotfiles/issues/796)) ([6c47989](https://github.com/mlorentedev/dotfiles/commit/6c47989642a165c8d9f24a18fa55a56ab5e70b6e))
* **hooks:** make the local hook stack executable on Windows and accept scoped commits ([#795](https://github.com/mlorentedev/dotfiles/issues/795)) ([17c7d40](https://github.com/mlorentedev/dotfiles/commit/17c7d4067aa2f9c1022748c82072096d52e6a5d3)), closes [#794](https://github.com/mlorentedev/dotfiles/issues/794)
* **hooks:** resolve local hooks through the shared git dir ([#805](https://github.com/mlorentedev/dotfiles/issues/805)) ([6873eca](https://github.com/mlorentedev/dotfiles/commit/6873ecaf2eeb64b2d71353439a1925fc73603c76))
* **spec-gate:** count a mandated archive as the Discipline Gate's spec touch ([#808](https://github.com/mlorentedev/dotfiles/issues/808)) ([c4fac9a](https://github.com/mlorentedev/dotfiles/commit/c4fac9a895e4a1a68a37c3f005b55d0895c594cb))
* **spec-gate:** scan the PR body for closing keywords as markdown, not as text ([#815](https://github.com/mlorentedev/dotfiles/issues/815)) ([47a9ab2](https://github.com/mlorentedev/dotfiles/commit/47a9ab2f9f530e5334acd9d75458acfb418fb81d))
* **spec:** make the agent-tag pre-flight match what the tooling emits ([#814](https://github.com/mlorentedev/dotfiles/issues/814)) ([4917986](https://github.com/mlorentedev/dotfiles/commit/49179861f64d067b1f5897bfc9037da4a84ecdfb))
* **test:** test the committed tree, not the deploy mirror ([#799](https://github.com/mlorentedev/dotfiles/issues/799)) ([7381860](https://github.com/mlorentedev/dotfiles/commit/7381860ced619e828c85c9daec53d9f88d27aef1)), closes [#794](https://github.com/mlorentedev/dotfiles/issues/794)

## [0.32.2](https://github.com/mlorentedev/dotfiles/compare/v0.32.1...v0.32.2) (2026-08-07)


### Bug Fixes

* **hooks:** run pre-commit gates that a global core.hooksPath made uninstallable ([#765](https://github.com/mlorentedev/dotfiles/issues/765)) ([90c4409](https://github.com/mlorentedev/dotfiles/commit/90c440914c74f14cdee91e755b8b6cf9749df9d8)), closes [#748](https://github.com/mlorentedev/dotfiles/issues/748)
* **install-dotf:** swap the binary atomically so upgrades survive a live dotf ([#760](https://github.com/mlorentedev/dotfiles/issues/760)) ([c55dfd7](https://github.com/mlorentedev/dotfiles/commit/c55dfd7aa6f3270204d72e5f8cc3cafd9c78546f)), closes [#750](https://github.com/mlorentedev/dotfiles/issues/750)
* **pwsh:** clear the built-in aliases that made four profile functions dead ([#763](https://github.com/mlorentedev/dotfiles/issues/763)) ([30e2c8e](https://github.com/mlorentedev/dotfiles/commit/30e2c8e48fa348a65193587936ea7f2e01f0a721)), closes [#745](https://github.com/mlorentedev/dotfiles/issues/745)
* **shell-profile:** keep the profiled shell's exit status from aborting the run ([#762](https://github.com/mlorentedev/dotfiles/issues/762)) ([2ee1105](https://github.com/mlorentedev/dotfiles/commit/2ee1105211f87fb7d53cc3438634d14113e50bb5)), closes [#746](https://github.com/mlorentedev/dotfiles/issues/746)

## [0.32.1](https://github.com/mlorentedev/dotfiles/compare/v0.32.0...v0.32.1) (2026-08-06)


### Bug Fixes

* **pi:** seed settings.json instead of overwriting it on every setup run ([#756](https://github.com/mlorentedev/dotfiles/issues/756)) ([aae1376](https://github.com/mlorentedev/dotfiles/commit/aae13761ad5c25ebe68e71a1dd4e4e34451706b3))

## [0.32.0](https://github.com/mlorentedev/dotfiles/compare/v0.31.7...v0.32.0) (2026-08-05)


### Features

* **pi:** add the deepseek-v4-flash-0731 model and cross-file config guards ([#749](https://github.com/mlorentedev/dotfiles/issues/749)) ([d2ded93](https://github.com/mlorentedev/dotfiles/commit/d2ded93dec2e9688965a74bdda1a47aeba318a4b))

## [0.31.7](https://github.com/mlorentedev/dotfiles/compare/v0.31.6...v0.31.7) (2026-08-05)


### Bug Fixes

* **shell:** move the gemini prompt helper out of the git-plugin alias namespace ([#744](https://github.com/mlorentedev/dotfiles/issues/744)) ([f7232d3](https://github.com/mlorentedev/dotfiles/commit/f7232d31413ac99bdd313ab3fea0e7e66bc465e3))

## [0.31.6](https://github.com/mlorentedev/dotfiles/compare/v0.31.5...v0.31.6) (2026-07-14)


### Bug Fixes

* **doctor:** install GUARD-001 memory-sink hooks on Windows + fix agy abs-path check ([#741](https://github.com/mlorentedev/dotfiles/issues/741)) ([2b58ebf](https://github.com/mlorentedev/dotfiles/commit/2b58ebf9a133e7383ba8213841101c6e4c22ffe5)), closes [#691](https://github.com/mlorentedev/dotfiles/issues/691)
* **mem:** own the Claude project-key encoding in Go so the Windows twins can't drift ([#739](https://github.com/mlorentedev/dotfiles/issues/739)) ([c4f1a7c](https://github.com/mlorentedev/dotfiles/commit/c4f1a7c4427f13a8e31f5e4bdece7b03bf134db3))

## [0.31.5](https://github.com/mlorentedev/dotfiles/compare/v0.31.4...v0.31.5) (2026-07-10)


### Bug Fixes

* **doctor:** resolve contract/versions repo-first via a shared resolver ([#736](https://github.com/mlorentedev/dotfiles/issues/736)) ([54fe5ca](https://github.com/mlorentedev/dotfiles/commit/54fe5ca75a7f40de02185a08e139be501b152f71))
* **env:** seed machine.json so update/mem resolve the real checkout ([#732](https://github.com/mlorentedev/dotfiles/issues/732)) ([374d816](https://github.com/mlorentedev/dotfiles/commit/374d81680e87c22101b2d3c06f30d27428845502))

## [0.31.4](https://github.com/mlorentedev/dotfiles/compare/v0.31.3...v0.31.4) (2026-07-09)


### Bug Fixes

* **setup:** refuse the in-place install layout that corrupts the checkout ([#726](https://github.com/mlorentedev/dotfiles/issues/726)) ([d21b01d](https://github.com/mlorentedev/dotfiles/commit/d21b01d93c755c99e6dc568f3e353b23496ace6c)), closes [#695](https://github.com/mlorentedev/dotfiles/issues/695)
* **spec-gate:** fail closed and close the three SDD-gate bypass routes ([#716](https://github.com/mlorentedev/dotfiles/issues/716)) ([3565d4b](https://github.com/mlorentedev/dotfiles/commit/3565d4b1bd1feb176662615242adf08b62e7a9b6)), closes [#686](https://github.com/mlorentedev/dotfiles/issues/686)

## [0.31.3](https://github.com/mlorentedev/dotfiles/compare/v0.31.2...v0.31.3) (2026-07-09)


### Bug Fixes

* **setup:** stop setup writing into the checkout so dotf update keeps deploying ([#714](https://github.com/mlorentedev/dotfiles/issues/714)) ([c30ee07](https://github.com/mlorentedev/dotfiles/commit/c30ee077b76cf859507a8648506243a0c24c751f)), closes [#694](https://github.com/mlorentedev/dotfiles/issues/694)

## [0.31.2](https://github.com/mlorentedev/dotfiles/compare/v0.31.1...v0.31.2) (2026-07-08)


### Bug Fixes

* **secrets:** stop secret argv leaks and drop forbidden auto-merge in ops scripts ([#711](https://github.com/mlorentedev/dotfiles/issues/711)) ([11e0e3b](https://github.com/mlorentedev/dotfiles/commit/11e0e3b4fb4b228fc7c10403140f7e19612b6221))

## [0.31.1](https://github.com/mlorentedev/dotfiles/compare/v0.31.0...v0.31.1) (2026-07-08)


### Bug Fixes

* **secrets:** correct id casing at every dotf secrets show call site ([#709](https://github.com/mlorentedev/dotfiles/issues/709)) ([b354408](https://github.com/mlorentedev/dotfiles/commit/b354408ac2dee917f90b4b445c59388277f4436c)), closes [#698](https://github.com/mlorentedev/dotfiles/issues/698)
* **secrets:** retire env-mapping.conf again, guard against resurrection ([#705](https://github.com/mlorentedev/dotfiles/issues/705)) ([8e58f8b](https://github.com/mlorentedev/dotfiles/commit/8e58f8b4e1c5755c08f4eb64a2a40dabd6b7d535))

## [0.31.0](https://github.com/mlorentedev/dotfiles/compare/v0.30.0...v0.31.0) (2026-07-01)


### Features

* **cli:** add dotf update, porting the self-deploy twins to Go ([#667](https://github.com/mlorentedev/dotfiles/issues/667)) ([ccc3189](https://github.com/mlorentedev/dotfiles/commit/ccc31893077d7fa51bd19c37c5c886561f4ff6a8)), closes [#496](https://github.com/mlorentedev/dotfiles/issues/496)
* **secrets:** add dotf secrets backup DR escrow (ADR-028) ([#661](https://github.com/mlorentedev/dotfiles/issues/661)) ([4683064](https://github.com/mlorentedev/dotfiles/commit/4683064aeb67ace8f4afdf3f694af2035a5a9315))
* **secrets:** verify age root-of-trust in doctor and declare key discovery vars ([#663](https://github.com/mlorentedev/dotfiles/issues/663)) ([2f52f00](https://github.com/mlorentedev/dotfiles/commit/2f52f007438a88421629ac0d8ee4562f123cd5af))

## [0.30.0](https://github.com/mlorentedev/dotfiles/compare/v0.29.0...v0.30.0) (2026-06-28)


### Features

* **git-hooks:** self-assign the linked issue at branch pickup ([#653](https://github.com/mlorentedev/dotfiles/issues/653)) ([ef5db02](https://github.com/mlorentedev/dotfiles/commit/ef5db0254fa832d49d580afb781f6455d5e2d30a))


### Bug Fixes

* **secrets:** apply file-secret mode + materialize atomically ([#612](https://github.com/mlorentedev/dotfiles/issues/612) B2/B4) ([#650](https://github.com/mlorentedev/dotfiles/issues/650)) ([c95f460](https://github.com/mlorentedev/dotfiles/commit/c95f460a54bf0a7ebd50177f213ee35ef6613a89))
* **secrets:** parse-time guards — var uniqueness + name/path validation ([#612](https://github.com/mlorentedev/dotfiles/issues/612) B1/B5) ([#651](https://github.com/mlorentedev/dotfiles/issues/651)) ([db4f8aa](https://github.com/mlorentedev/dotfiles/commit/db4f8aa2d0a08426756ece829a84b93eae5f4d6e))

## [0.29.0](https://github.com/mlorentedev/dotfiles/compare/v0.28.0...v0.29.0) (2026-06-28)


### Features

* **secrets:** opt-in github-token liveness check before sync ci upload ([#639](https://github.com/mlorentedev/dotfiles/issues/639)) ([906fe21](https://github.com/mlorentedev/dotfiles/commit/906fe2175d61a74267426034d3cef059b0aebd7b))


### Bug Fixes

* **secrets:** resolve registry from the checkout SSOT, not the deployed copy ([#635](https://github.com/mlorentedev/dotfiles/issues/635)) ([#636](https://github.com/mlorentedev/dotfiles/issues/636)) ([cf8a9e1](https://github.com/mlorentedev/dotfiles/commit/cf8a9e1001e0c8c413a98ae64886a59bb4cd64e3))
* **secrets:** resolve the age store checkout-first too + fix ADR-029 collision ([#642](https://github.com/mlorentedev/dotfiles/issues/642)) ([9a51faf](https://github.com/mlorentedev/dotfiles/commit/9a51faf9852d98d976bdad38fc12a27a3f93174f))
* **secrets:** surface age's stderr on decrypt failure, not 'exit status 1' ([#644](https://github.com/mlorentedev/dotfiles/issues/644)) ([cf9323a](https://github.com/mlorentedev/dotfiles/commit/cf9323ae7e516fba94be7bd799421896fcb36b3b))
* **setup:** complete claude-mem retirement — strip the marketplace from settings.json ([#645](https://github.com/mlorentedev/dotfiles/issues/645)) ([8afe9f5](https://github.com/mlorentedev/dotfiles/commit/8afe9f5c100fc3855a4002f349207f3633e756fc))
* **setup:** repair shellcheck install — versioned asset URL + fail-loud curl ([#648](https://github.com/mlorentedev/dotfiles/issues/648)) ([9fec75c](https://github.com/mlorentedev/dotfiles/commit/9fec75caa69cf5e2c0268a15db582acb9c25e25f))

## [0.28.0](https://github.com/mlorentedev/dotfiles/compare/v0.27.0...v0.28.0) (2026-06-26)


### Features

* **secrets:** dotf secrets sync ci (backend-agnostic Actions materialization) ([#632](https://github.com/mlorentedev/dotfiles/issues/632)) ([e34ab89](https://github.com/mlorentedev/dotfiles/commit/e34ab89b95d3ce92ab126196be2f47c8359de209))

## [0.27.0](https://github.com/mlorentedev/dotfiles/compare/v0.26.0...v0.27.0) (2026-06-26)


### Features

* **secrets:** add `dotf secrets migrate` (age→bw cutover, parity-gated) ([#627](https://github.com/mlorentedev/dotfiles/issues/627)) ([0453fc2](https://github.com/mlorentedev/dotfiles/commit/0453fc24af9094550bbcb14fa6405d7edc9b9cc3))


### Bug Fixes

* **mem:** resolve a real bash, not the System32 WSL launcher, for vault-health ([#629](https://github.com/mlorentedev/dotfiles/issues/629)) ([1a664d4](https://github.com/mlorentedev/dotfiles/commit/1a664d4c3fc6f2a49069376b44feca00d517e517))

## [0.26.0](https://github.com/mlorentedev/dotfiles/compare/v0.25.0...v0.26.0) (2026-06-26)


### Features

* **secrets:** reorganize registry — one entry per env var + bw: targets ([#624](https://github.com/mlorentedev/dotfiles/issues/624)) ([c2ddf95](https://github.com/mlorentedev/dotfiles/commit/c2ddf95091ecc7ff57e2cac717ed4275cd010e52))

## [0.25.0](https://github.com/mlorentedev/dotfiles/compare/v0.24.0...v0.25.0) (2026-06-26)


### Features

* **secrets:** add `dotf secrets set` idempotent bw write command ([#621](https://github.com/mlorentedev/dotfiles/issues/621)) ([3c6a7ab](https://github.com/mlorentedev/dotfiles/commit/3c6a7ab52663fff3ca305ba97c905d52d63acb5c))

## [0.24.0](https://github.com/mlorentedev/dotfiles/compare/v0.23.0...v0.24.0) (2026-06-26)


### Features

* **ci:** lint bats [@test](https://github.com/test) names to stop silent-skipped tests ([#619](https://github.com/mlorentedev/dotfiles/issues/619)) ([614f4ec](https://github.com/mlorentedev/dotfiles/commit/614f4ec99b207b3ef9dd744f892099c4e681e05b))
* **secrets:** add BWWriter (bw write seam, read-modify-write) ([#620](https://github.com/mlorentedev/dotfiles/issues/620)) ([f936649](https://github.com/mlorentedev/dotfiles/commit/f9366496b04ba868c8b122224fab07a51a47c97e))
* **secrets:** add SetBackendBW registry mutation primitive ([#617](https://github.com/mlorentedev/dotfiles/issues/617)) ([c6f659f](https://github.com/mlorentedev/dotfiles/commit/c6f659f77f461c603ab734a21bfa2a2ae17bb5f7))

## [0.23.0](https://github.com/mlorentedev/dotfiles/compare/v0.22.0...v0.23.0) (2026-06-26)


### Features

* **secrets:** add `dotf secrets verify` health check ([#616](https://github.com/mlorentedev/dotfiles/issues/616)) ([c706f87](https://github.com/mlorentedev/dotfiles/commit/c706f87df9b3956c1ebe829601a39c41f4105b30))


### Bug Fixes

* **secrets:** make resolution fail loud instead of silently empty ([#613](https://github.com/mlorentedev/dotfiles/issues/613)) ([411c7c0](https://github.com/mlorentedev/dotfiles/commit/411c7c09ed373dd99509b262ad5f89f1b6dcc52f))

## [0.22.0](https://github.com/mlorentedev/dotfiles/compare/v0.21.1...v0.22.0) (2026-06-26)


### Features

* **harness:** agnostic agent-skill presence by uniform injection ([#607](https://github.com/mlorentedev/dotfiles/issues/607)) ([3d61c2c](https://github.com/mlorentedev/dotfiles/commit/3d61c2cbb0456948a9eae157405a3a321e857df8)), closes [#559](https://github.com/mlorentedev/dotfiles/issues/559)
* **secrets:** add the Bitwarden backend resolver to dotf secrets ([#606](https://github.com/mlorentedev/dotfiles/issues/606)) ([736273b](https://github.com/mlorentedev/dotfiles/commit/736273b8922c75539c913df537b289e52520f87a))
* **secrets:** strip backend unlock credentials from the run child env ([#610](https://github.com/mlorentedev/dotfiles/issues/610)) ([8b56f1f](https://github.com/mlorentedev/dotfiles/commit/8b56f1f1205789ac28aee35305cc08bb9f42d129))

## [0.21.1](https://github.com/mlorentedev/dotfiles/compare/v0.21.0...v0.21.1) (2026-06-25)


### Bug Fixes

* **spec-gate:** exclude Go *_test.go from the production-LOC count ([#603](https://github.com/mlorentedev/dotfiles/issues/603)) ([5d726ce](https://github.com/mlorentedev/dotfiles/commit/5d726ce166b9a1e2f4dd79bdea112f6eebe69b60)), closes [#517](https://github.com/mlorentedev/dotfiles/issues/517)

## [0.21.0](https://github.com/mlorentedev/dotfiles/compare/v0.20.0...v0.21.0) (2026-06-25)


### Features

* **secrets:** retire the deploy-time shell twins and env-mapping.conf ([#601](https://github.com/mlorentedev/dotfiles/issues/601)) ([83476da](https://github.com/mlorentedev/dotfiles/commit/83476da2bc325a21eda64b3c8369a1f5876dfcf1))

## [0.20.0](https://github.com/mlorentedev/dotfiles/compare/v0.19.1...v0.20.0) (2026-06-25)


### Features

* **secrets:** add dotf secrets render and wire setups off the shell twins ([#596](https://github.com/mlorentedev/dotfiles/issues/596)) ([5cc62f3](https://github.com/mlorentedev/dotfiles/commit/5cc62f3287d397a6279478937838ee03d7b9a499))

## [0.19.1](https://github.com/mlorentedev/dotfiles/compare/v0.19.0...v0.19.1) (2026-06-25)


### Bug Fixes

* **secrets:** deploy secrets/registry.yaml in setup so dotf secrets works ([#591](https://github.com/mlorentedev/dotfiles/issues/591)) ([bc33bbc](https://github.com/mlorentedev/dotfiles/commit/bc33bbcd4aa1d1130e93a04fabefd5debe03cfc6))

## [0.19.0](https://github.com/mlorentedev/dotfiles/compare/v0.18.0...v0.19.0) (2026-06-25)


### Features

* **secrets:** resolve nan-* scripts' NAN_API_KEY via dotf secrets show ([#588](https://github.com/mlorentedev/dotfiles/issues/588)) ([3245be5](https://github.com/mlorentedev/dotfiles/commit/3245be5e449b0a3749c452d1c76f22638fe85163))

## [0.18.0](https://github.com/mlorentedev/dotfiles/compare/v0.17.1...v0.18.0) (2026-06-25)


### Features

* **doctor:** repair auto-memory junction + OS-aware env-contract checks (HARNESS-040) ([#576](https://github.com/mlorentedev/dotfiles/issues/576)) ([6d2627c](https://github.com/mlorentedev/dotfiles/commit/6d2627cb2ac6aeb2da1b93ceeed6a150674025d7)), closes [#551](https://github.com/mlorentedev/dotfiles/issues/551)
* **secrets:** stop ambient secret export; wrap AI CLIs via dotf secrets run ([#581](https://github.com/mlorentedev/dotfiles/issues/581)) ([e957c4f](https://github.com/mlorentedev/dotfiles/commit/e957c4f110732ba0503020a3db6d0e7433a9102b)), closes [#493](https://github.com/mlorentedev/dotfiles/issues/493)

## [0.17.1](https://github.com/mlorentedev/dotfiles/compare/v0.17.0...v0.17.1) (2026-06-24)


### Bug Fixes

* **vault:** scaffold number-free context/roadmap filenames (KPM-P) ([#572](https://github.com/mlorentedev/dotfiles/issues/572)) ([4c08b72](https://github.com/mlorentedev/dotfiles/commit/4c08b7270a0a83a74ed6d4261fee46f2f7058fdf))

## [0.17.0](https://github.com/mlorentedev/dotfiles/compare/v0.16.0...v0.17.0) (2026-06-24)


### Features

* **mem:** assemble the Claude session-start adapter + golden gate (CLI-025) ([#569](https://github.com/mlorentedev/dotfiles/issues/569)) ([dd95039](https://github.com/mlorentedev/dotfiles/commit/dd95039855db89a31709520e2be562f979c31cd5))
* **mem:** Claude session-start injectors (CLI-025) ([#566](https://github.com/mlorentedev/dotfiles/issues/566)) ([ecfff9d](https://github.com/mlorentedev/dotfiles/commit/ecfff9d256ac22cd1437cfde179d224f1be693da))
* **mem:** cut over the SessionStart hook to dotf mem session-start, delete the shell cluster (CLI-025) ([#570](https://github.com/mlorentedev/dotfiles/issues/570)) ([0a18373](https://github.com/mlorentedev/dotfiles/commit/0a18373a37682c58f64fac9d3b555cc1dd430903))
* **memlink:** OS-agnostic vault-&gt;memory link primitive (CLI-025) ([#557](https://github.com/mlorentedev/dotfiles/issues/557)) ([edec57c](https://github.com/mlorentedev/dotfiles/commit/edec57c5607deb2e49c3edc1db3545e813b6712c))

## [0.16.0](https://github.com/mlorentedev/dotfiles/compare/v0.15.0...v0.16.0) (2026-06-24)


### Features

* **doctor:** provision knowledge-vault git hooks (OPS-016) ([#553](https://github.com/mlorentedev/dotfiles/issues/553)) ([ca02475](https://github.com/mlorentedev/dotfiles/commit/ca0247542ff484a4b4ab0f5afd69de0771e0521d))
* **mem:** port session-brief agnostic core to dotf mem session-start (CLI-025) ([#554](https://github.com/mlorentedev/dotfiles/issues/554)) ([0cadeac](https://github.com/mlorentedev/dotfiles/commit/0cadeac3c91943535e9d9172b1bf5fe7708ebdbf))

## [0.15.0](https://github.com/mlorentedev/dotfiles/compare/v0.14.1...v0.15.0) (2026-06-24)


### Features

* **mem:** port session-handoff to dotf mem session-end, delete shell twins (CLI-025) ([#546](https://github.com/mlorentedev/dotfiles/issues/546)) ([75c40ea](https://github.com/mlorentedev/dotfiles/commit/75c40eae947666270f97f8cef17ccb94d57ef41d))

## [0.14.1](https://github.com/mlorentedev/dotfiles/compare/v0.14.0...v0.14.1) (2026-06-23)


### Bug Fixes

* **session-handoff:** write records to the project folder, not 00_meta/sessions ([#542](https://github.com/mlorentedev/dotfiles/issues/542)) ([1a185b1](https://github.com/mlorentedev/dotfiles/commit/1a185b1e8a46cbeff39a027b28f920d0efa22227))

## [0.14.0](https://github.com/mlorentedev/dotfiles/compare/v0.13.0...v0.14.0) (2026-06-22)


### Features

* **tools:** dotf tools install — download + checksum-verify catalog tools (CLI-029) ([#526](https://github.com/mlorentedev/dotfiles/issues/526)) ([9d6f2ed](https://github.com/mlorentedev/dotfiles/commit/9d6f2ed28b7c7e55779c6b5450120e938d4b6a08))

## [0.13.0](https://github.com/mlorentedev/dotfiles/compare/v0.12.0...v0.13.0) (2026-06-21)


### Features

* **doctor:** port healthcheck section 4 deployed-config checks ([#522](https://github.com/mlorentedev/dotfiles/issues/522)) ([7f9f3b6](https://github.com/mlorentedev/dotfiles/commit/7f9f3b66ad8caf47283e36c025811d5293afc484)), closes [#509](https://github.com/mlorentedev/dotfiles/issues/509)

## [0.12.0](https://github.com/mlorentedev/dotfiles/compare/v0.11.0...v0.12.0) (2026-06-21)


### Features

* **doctor:** port repo↔deploy-dir drift check (CLI-019 PR-A) ([#513](https://github.com/mlorentedev/dotfiles/issues/513)) ([699e34c](https://github.com/mlorentedev/dotfiles/commit/699e34c17e1694c0ccbc02ab9a998aa142014cac))

## [0.11.0](https://github.com/mlorentedev/dotfiles/compare/v0.10.0...v0.11.0) (2026-06-21)


### Features

* **bash:** opt-in userspace ssh-agent autoload ([#507](https://github.com/mlorentedev/dotfiles/issues/507)) ([cbe1c78](https://github.com/mlorentedev/dotfiles/commit/cbe1c787d9cb735e8deba2a0c24d0485f53ac72d))
* **tools:** declarative package catalog + dotf tools list (CLI-029 pilot) ([#508](https://github.com/mlorentedev/dotfiles/issues/508)) ([8332630](https://github.com/mlorentedev/dotfiles/commit/8332630260d9c4c8cb671f77bc09a33b985c4163))


### Bug Fixes

* **harness:** CRLF-robust refresh + reconcile skill records with vault SSOT ([#511](https://github.com/mlorentedev/dotfiles/issues/511)) ([7328965](https://github.com/mlorentedev/dotfiles/commit/7328965e6981a5a1db8c6f4b063c620367ceed30))

## [0.10.0](https://github.com/mlorentedev/dotfiles/compare/v0.9.4...v0.10.0) (2026-06-21)


### Features

* **doctor:** port the Orca Copilot hook (DX-006) check into dotf doctor ([#505](https://github.com/mlorentedev/dotfiles/issues/505)) ([3701936](https://github.com/mlorentedev/dotfiles/commit/3701936bedf68df6d3ddf4c86119d3999620ed30))
* **handoff:** cache-stable block placement + agnostic lessons-staleness signal ([#502](https://github.com/mlorentedev/dotfiles/issues/502)) ([9de802b](https://github.com/mlorentedev/dotfiles/commit/9de802b6a760019bccf90bd51ec28d13adf748c5))
* **ssh:** add *-ext bastion aliases for off-LAN fleet access ([#503](https://github.com/mlorentedev/dotfiles/issues/503)) ([260008f](https://github.com/mlorentedev/dotfiles/commit/260008f1fe4de6003e625056ac2c4cd3b3f6d4c4))

## [0.9.4](https://github.com/mlorentedev/dotfiles/compare/v0.9.3...v0.9.4) (2026-06-20)


### Bug Fixes

* **profile:** resolve nan-debug.sh via DOTFILES_REPO_DIR, not a hardcoded literal ([#482](https://github.com/mlorentedev/dotfiles/issues/482)) ([2a23355](https://github.com/mlorentedev/dotfiles/commit/2a2335562cf5a292f59a4794b911c4c596a056fc))
* **shell:** rename gp-&gt;gpr (collision) and source utils.sh declaratively ([#484](https://github.com/mlorentedev/dotfiles/issues/484)) ([e1c0090](https://github.com/mlorentedev/dotfiles/commit/e1c00900a1dd93988020962042f51f09e98bdc35))

## [0.9.3](https://github.com/mlorentedev/dotfiles/compare/v0.9.2...v0.9.3) (2026-06-20)


### Bug Fixes

* **setup:** enforce versions.conf pins as minimums, not presence or exact match ([#480](https://github.com/mlorentedev/dotfiles/issues/480)) ([a9fdd60](https://github.com/mlorentedev/dotfiles/commit/a9fdd60f8872e6e60642806ff23577d083536913))

## [0.9.2](https://github.com/mlorentedev/dotfiles/compare/v0.9.1...v0.9.2) (2026-06-20)


### Bug Fixes

* **setup:** align Linux deploy strategy with Windows always-overwrite ([#476](https://github.com/mlorentedev/dotfiles/issues/476)) ([d653db3](https://github.com/mlorentedev/dotfiles/commit/d653db30608695ada867e36803022d57aafa919b))
* **setup:** correct Compare-Object -SyncId errors and add pi version drift check ([#474](https://github.com/mlorentedev/dotfiles/issues/474)) ([58cd9e3](https://github.com/mlorentedev/dotfiles/commit/58cd9e3389b3c735754c8dd571ee0c5a479ff3db))

## [0.9.1](https://github.com/mlorentedev/dotfiles/compare/v0.9.0...v0.9.1) (2026-06-20)


### Bug Fixes

* **ci:** add PRs to bitácora board via gh CLI ([#470](https://github.com/mlorentedev/dotfiles/issues/470)) ([01d7bb5](https://github.com/mlorentedev/dotfiles/commit/01d7bb5e9c7c17724a3aec33f0aebb5b483262f4))
* **ci:** rewrite add-to-project PR step with GraphQL API ([#473](https://github.com/mlorentedev/dotfiles/issues/473)) ([f2312a5](https://github.com/mlorentedev/dotfiles/commit/f2312a525a7055f7a5dcc90a6d3cf5cf6068387f))

## [0.9.0](https://github.com/mlorentedev/dotfiles/compare/v0.8.1...v0.9.0) (2026-06-20)


### Features

* **setup:** autostart the hive daemon via Startup folder when Task Scheduler is blocked ([#467](https://github.com/mlorentedev/dotfiles/issues/467)) ([12c8d50](https://github.com/mlorentedev/dotfiles/commit/12c8d502350d15d6262d11c5ca47f5034b7469fc))

## [0.8.1](https://github.com/mlorentedev/dotfiles/compare/v0.8.0...v0.8.1) (2026-06-20)


### Bug Fixes

* **session-start:** match Claude Code's path encoding for the memory junction ([#466](https://github.com/mlorentedev/dotfiles/issues/466)) ([298fb60](https://github.com/mlorentedev/dotfiles/commit/298fb602fe90fd1647b7f573efb88923973a59eb))
* **setup:** install Bun on Windows so the claude-mem worker can start ([#464](https://github.com/mlorentedev/dotfiles/issues/464)) ([2785235](https://github.com/mlorentedev/dotfiles/commit/27852355ca610ff400aefb904faec5adcd82b1e2))

## [0.8.0](https://github.com/mlorentedev/dotfiles/compare/v0.7.0...v0.8.0) (2026-06-19)


### Features

* **harness:** harden ADR-025 cross-machine path resolution end-to-end (HARNESS-027, [#457](https://github.com/mlorentedev/dotfiles/issues/457)) ([#458](https://github.com/mlorentedev/dotfiles/issues/458)) ([6594288](https://github.com/mlorentedev/dotfiles/commit/6594288720d4724ec33ff79c3d7ca831f31554d4))


### Bug Fixes

* **windows:** re-apply Orca Copilot hook fix idempotently (DX-006) ([#456](https://github.com/mlorentedev/dotfiles/issues/456)) ([6f045a4](https://github.com/mlorentedev/dotfiles/commit/6f045a43da42bd90d95263ec5ddd4b574bc3f6d1))

## [0.7.0](https://github.com/mlorentedev/dotfiles/compare/v0.6.0...v0.7.0) (2026-06-19)


### Features

* **setup:** install dotf from the published release binary on Windows (WIN-006, [#451](https://github.com/mlorentedev/dotfiles/issues/451)) ([#453](https://github.com/mlorentedev/dotfiles/issues/453)) ([1f22769](https://github.com/mlorentedev/dotfiles/commit/1f227693e67501a5b0a6f6ae8f5a6873a6aa943a))

## [0.6.0](https://github.com/mlorentedev/dotfiles/compare/v0.5.1...v0.6.0) (2026-06-19)


### Features

* **cli:** cross-machine path resolution via dotf env generate (CLI-016, [#445](https://github.com/mlorentedev/dotfiles/issues/445)) ([#447](https://github.com/mlorentedev/dotfiles/issues/447)) ([60d120b](https://github.com/mlorentedev/dotfiles/commit/60d120bdd4d10eb75368b4fb6abcf560df57af48))
* **setup:** wire resolved vault path into setup + hive daemon (HARNESS-024, [#446](https://github.com/mlorentedev/dotfiles/issues/446)) ([#448](https://github.com/mlorentedev/dotfiles/issues/448)) ([4d8ce18](https://github.com/mlorentedev/dotfiles/commit/4d8ce184af2fe7bf1d4f3307e32b649be7ef0119))

## [0.5.1](https://github.com/mlorentedev/dotfiles/compare/v0.5.0...v0.5.1) (2026-06-18)


### Bug Fixes

* **setup:** install pi into ~/.local so GUI/ADE launchers resolve it ([#440](https://github.com/mlorentedev/dotfiles/issues/440)) ([f22e425](https://github.com/mlorentedev/dotfiles/commit/f22e425e47e6da45dcbfda9da3b4434ab3f693aa)), closes [#426](https://github.com/mlorentedev/dotfiles/issues/426)

## [0.5.0](https://github.com/mlorentedev/dotfiles/compare/v0.4.0...v0.5.0) (2026-06-18)


### Features

* **doctor:** detect expiring or invalid GitHub PATs before they break CI ([#427](https://github.com/mlorentedev/dotfiles/issues/427)) ([52695f3](https://github.com/mlorentedev/dotfiles/commit/52695f32e1de33b46f1e24a3f90322fb6b95d7db)), closes [#422](https://github.com/mlorentedev/dotfiles/issues/422)


### Bug Fixes

* **ci:** deterministic Windows tool install — age, eza, zoxide (BUG-025/024) ([#425](https://github.com/mlorentedev/dotfiles/issues/425)) ([fdb27f8](https://github.com/mlorentedev/dotfiles/commit/fdb27f854ed2dfb16d937975c5ff21d0f978d0aa))
* **doctor:** resolve PAT from any mapped env alias, not just the first ([#429](https://github.com/mlorentedev/dotfiles/issues/429)) ([add7d1d](https://github.com/mlorentedev/dotfiles/commit/add7d1de3104fd61bafd15fb523fc6d586ae5b63))

## [0.4.0](https://github.com/mlorentedev/dotfiles/compare/v0.3.0...v0.4.0) (2026-06-17)


### Features

* **ci:** adopt release-please (version + changelog + tag automation) ([#416](https://github.com/mlorentedev/dotfiles/issues/416)) ([a17c917](https://github.com/mlorentedev/dotfiles/commit/a17c917ba66d08b9d75932c1ff7291b963430590)), closes [#369](https://github.com/mlorentedev/dotfiles/issues/369)
* **guard:** complete GUARD-001 single-sink (gitignore, global install, AGENTS.md) ([#415](https://github.com/mlorentedev/dotfiles/issues/415)) ([a4cd005](https://github.com/mlorentedev/dotfiles/commit/a4cd005964480c87de36f0f12c2dfd76f6399068))
* **session-start:** extract agent-agnostic session-brief core (ADR-023) ([#413](https://github.com/mlorentedev/dotfiles/issues/413)) ([5f34eee](https://github.com/mlorentedev/dotfiles/commit/5f34eeeef18e28d81f3cd15baa82a1d5f1c6221c))


### Bug Fixes

* **ci:** point release-please at the existing RELEASE_TOKEN secret ([#419](https://github.com/mlorentedev/dotfiles/issues/419)) ([242f6d1](https://github.com/mlorentedev/dotfiles/commit/242f6d1ff1a61ca65ad72041f4b6ad634e8fdd2c)), closes [#369](https://github.com/mlorentedev/dotfiles/issues/369)
* **guard:** deploy the memory-sink dispatcher + wire core.hooksPath ([#418](https://github.com/mlorentedev/dotfiles/issues/418)) ([#420](https://github.com/mlorentedev/dotfiles/issues/420)) ([3d551a6](https://github.com/mlorentedev/dotfiles/commit/3d551a690b5fc0a941953276c6246ebbce9ce493))

## Changelog

Maintained by [release-please](https://github.com/googleapis/release-please) from Conventional Commits. Do not edit by hand.

## Features

- 2026-05-17: feat(AI-012): port Claude skills to OpenCode commands (d326954)
- 2026-05-17: feat(aliases): add oclog for live opencode log tailing (91ebdf7)
- 2026-05-16: feat(agents-md): salvage MCP rules from stale refactor branches (c6e049b)
- 2026-05-16: feat(AI-011): bootstrap opencode + canonical AGENTS.md migration (0d7fed8)
- 2026-05-15: feat(doctor): SessionStart silent doctor + binary version pinning (4e9798a)
- 2026-05-15: feat(doctor): declarative env contract + doctor.sh/ps1 with --check/--fix (d6ced62)
- 2026-05-14: feat(scripts): add init-repo-standards generator (SDD-010) (adfd638)
- 2026-05-14: feat(scripts): SessionStart hook surfaces repo specs/ state (SDD-016) (eb32d75)
- 2026-05-14: feat(scripts): vault working-tree integrity check at session start (SDD-017) (c39e6f6)
- 2026-05-14: feat(scripts): add init-repo-agents bootstrap for AGENTS.md (SDD-013) (a7f9b9a)
- 2026-05-13: feat(claude-md): add claude-mem MCP rules + dual-memory protocol pointer (c8a93f6)
- 2026-05-13: feat(setup): auto-link vault-hosted skills into ~/.claude/skills/ (2137ffa)
- 2026-05-13: feat(scripts): add init-spec + archive-spec for SDD per-feature workflow (d77021b)
- 2026-05-12: feat(scripts): opt-in shell startup profiling (8a83e1c)
- 2026-05-12: feat(scripts): add changelog-gen.sh + initial CHANGELOG.md (63d4a36)
- 2026-05-12: feat(scripts): add diff-check.sh to detect repo ↔ deploy-dir drift (3f7af6a)
- 2026-05-12: feat(tmux): add focus-events, vi visual-mode bindings, slower status refresh (c23ec99)
- 2026-05-11: feat(tmux): copy selection to system clipboard via xclip (fd361f6)
- 2026-05-11: feat(tmux): integrate tmux with versioned config and Linux install (239e715)
- 2026-05-08: feat(scripts): add claude-mem-heal for upstream v12/v13 packaging bugs (053bad8)
- 2026-03-29: feat: skills ecosystem overhaul — 23 to 17 skills, CSO audit, Standing Orders (61c4b38)
- 2026-03-27: feat: add obs-cli wrapper for Obsidian CLI (Linux + Windows) (91ba19d)
- 2026-03-26: feat: unified workflow protocol — area-agnostic CLAUDE.md, full vault entry in init-project, work SDK detection in session hooks (3884d4f)
- 2026-03-26: feat(hooks): auto-create memory junction/symlink on session start (999a478)
- 2026-03-26: feat(setup): bidirectional memory sync on Windows via junctions (ef53bd4)
- 2026-03-25: feat(ai,secrets): add engineering discipline rules, secrets reconciliation, cleanup (a9fb76a)
- 2026-03-24: feat(claude): add self-maintaining memory system (0204133)
- 2026-03-16: feat(setup): auto-install 10 developer tools on Linux and 7 on Windows (e1e4746)
- 2026-03-10: feat(ai): add aider integration with 3-tier OpenRouter model config (c515c69)
- 2026-03-07: feat(setup): add hive MCP server with auto-upgrade to both Linux and Windows (3ddc041)
- 2026-02-28: feat(ai): add kc / kca shortcuts for quick access as aliases (e153f7e)
- 2026-02-28: feat(ai): knowledge crystallization system — bash + PowerShell + auto-discovery (067ee84)
- 2026-02-27: feat(setup): auto-register Claude Code SessionStart hook (03cb64e)
- 2026-02-27: feat: add Claude Code SessionStart hook for vault health context (917c03b)
- 2026-02-27: feat: add vault-health.sh and integrate Obsidian CLI checks (fc85d1d)
- 2026-02-27: feat(shell): add obsidian alias with --no-sandbox for Linux AppImage (dfe4c37)
- 2026-02-26: feat(ci): add container-based integration test for setup-linux.sh (465fd2a)
- 2026-02-26: feat: add versions.conf and healthcheck.sh (P1 backlog) (72fdb29)
- 2026-02-26: feat(shell): standardize set -euo pipefail across standalone scripts (fd5ef7e)
- 2026-02-26: feat(ai): add auto-memory to Neural Hive context sync phase (b01af3e)
- 2026-02-26: feat: persist MCP servers globally and auto-memory via vault (46abe35)
- 2026-02-26: feat: persist MCP servers and auto-memory across machines (ffd56fa)
- 2026-02-23: feat(claude): add no Co-Authored-By policy to global CLAUDE.md (4befe55)
- 2026-02-23: feat: set nano as default UNIX editor (3fa1bea)
- 2026-02-22: feat(ai): implement neural hive protocol and standardize vault (6c32d26)
- 2026-02-22: feat: add PSScriptAnalyzer linting to CI for PowerShell scripts (34b94d5)
- 2026-02-22: feat: add PSScriptAnalyzer linting to CI for PowerShell scripts (4197b4d)
- 2026-02-22: feat: add secrets_show command and SSH config deployment (9712196)
- 2026-02-21: feat: add file-based secrets support for kubeconfig and multiline files (9da08d8)
- 2026-02-21: feat: add prd, qa-plan, prd-to-issues skills and automate plugin installation (e1a9ad9)
- 2026-02-21: feat: add prd skill for interactive requirements gathering (ce7263e)
- 2026-02-18: feat: auto-install claude-mem plugin in setup scripts (8af28a7)
- 2026-02-16: feat: apply Anthropic Claude Code best practices across project and global config (5bb8aae)
- 2026-02-16: feat: apply Anthropic Claude Code best practices across project and global config (0fd3557)
- 2026-02-13: feat: add POLLEX_API_KEY (4e2bafe)
- 2026-02-11: feat: add excalidraw MCP server registration to setup scripts (2148176)
- 2026-02-10: feat: auto-create python symlink in setup for version-agnostic command (37950fe)
- 2026-02-10: feat: upgrade Go to 1.26.0 and prepend tool paths for system override (83e938a)
- 2026-02-08: feat: add bun PATH to .bashrc and .zshrc for persistent installation (c7d7590)
- 2026-02-04: feat: add USB backup of secrets with VeraCrypt support (032965b)
- 2026-02-02: feat: add Windows PowerShell support and rename setup scripts   - Add setup-windows.ps1, powershell/profile.ps1, scripts/init-project.ps1   - Rename install.sh → setup-linux.sh, change claude-init → project-init   - Delete obsolete .bat files   - Update all documentation (5a568af)
- 2026-02-01: feat: consolidate AI configuration and implement Claude Code skills   - Refactor CLAUDE.md and GEMINI.md   - Restructure skills to official format (SKILL.md with YAML frontmatter)   - Add skills: audit, refactor, test, doc, docker   - Update install.sh to copy skill directories and extract Gemini prompts   - Update init-project.sh for new skill structure   - Add docs/AI.md with complete setup and workflow guide   - Clean up deprecated versioned files (6037ada)
- 2026-01-14: feat: implement dotfiles sync and enhance secrets management 	- Add `dotfiles-sync` workflow for local vs repo synchronization 	- Update `github-secrets-manager` to support uploading from `env-mapping.conf` 	- Add auto-sync capabilities to `secrets_add` and `secrets_rotate` 	- Register `PYPI_TOKEN` in secrets mapping 	- Update documentation and test suite (87eab20)
- 2026-01-14: feat: implement dotfiles sync and enhance secrets management 	- Add `dotfiles-sync` workflow for local vs repo synchronization 	- Update `github-secrets-manager` to support uploading from `env-mapping.conf` 	- Add auto-sync capabilities to `secrets_add` and `secrets_rotate` 	- Register `PYPI_TOKEN` in secrets mapping 	- Update documentation and test suite (290c362)
- 2026-01-11: feat: add secrets availability in bash based on encrypted files with a config file mapping, add test suite as part of precommit hook. (09235cb)
- 2025-11-19: feat: introduce Claude AI aliases and prompt function,  and restructure documentation. (6280f4c)
- 2025-11-19: feat: initial commit with Claude optimization (25e7324)
- 2025-11-18: feat: support gemini-cli with custom GEMINI.md and prompts for easy use common to all projects (5018898)
- 2025-03-23: feat: add env file path as input parameter to setup-gh-secrets (5a3f6c6)

## Bug Fixes

- 2026-05-17: fix(BUG-001): update integration test for detect-and-act Copilot logic (5dfed05)
- 2026-05-17: fix(BUG-001): correct Copilot verification + gate config on extension presence (22fe726)
- 2026-05-17: fix(tmux): pass truecolor through to ghostty (TERM=xterm-ghostty) (fa36796)
- 2026-05-16: fix(lint): replace em dash with ASCII in profile.ps1 OpenCode comment (9d284b9)
- 2026-05-16: fix(AI-011): update CLAUDE/GEMINI deployment marker after AGENTS.md migration (a875103)
- 2026-05-15: fix(setup-windows): replace em dash with ASCII to satisfy PSScriptAnalyzer (464eecf)
- 2026-05-15: fix(setup): idempotent claude plugin install to stop .claude.json truncation (0d805f9)
- 2026-05-15: fix(setup): stop deleting vault content via symlink follow in skill sync (022d535)
- 2026-05-15: fix(setup-linux): close mcp-servers.json + doctor.sh parity gaps from Windows-side work (2775aa8)
- 2026-05-15: fix(doctor): persist structural env vars in profiles + section summaries (3176804)
- 2026-05-15: fix(scripts): claude-mem-heal Windows parity (zod/v3 plugin bug) (1b4288b)
- 2026-05-15: fix(setup): idempotent MCP registration + self-healing scheduled task (4f8a2c6)
- 2026-05-15: fix(setup): self-healing SessionStart hook + LF gitattributes (closes #20) (b5a3f6c)
- 2026-05-13: fix(setup): use ASCII hyphen in Write-Warn to satisfy PSScriptAnalyzer (001efa3)
- 2026-05-12: fix(secrets): keep env, deployed, and repo in sync after every mutation (295b6f3)
- 2026-05-08: fix(scripts): remove unused cwd_slug local in claude-session-start (f030959)
- 2026-03-29: fix(ci): update skill count threshold from 18 to 15 after ecosystem overhaul (103ec41)
- 2026-03-29: fix(ci): guard crontab call for environments without cron (8262808)
- 2026-03-27: fix(ci): replace remaining non-ASCII chars in init-project.ps1 (27dc6af)
- 2026-03-26: fix(ssh): aws1 uses MagicDNS instead of hardcoded Tailscale IP (c6ab454)
- 2026-03-26: fix(ci): replace non-ASCII chars in init-project.ps1 to pass PSScriptAnalyzer (60417ce)
- 2026-03-25: fix(setup): always deploy AI config regardless of CLI presence (c3be688)
- 2026-03-25: fix(tests): skip Gemini integration tests when CLI not installed (13c07a1)
- 2026-03-25: fix(tests): update healthcheck section count from 7 to 8 (f2c1c13)
- 2026-03-22: fix(ssh): sync config with live host inventory (14edd90)
- 2026-03-18: fix(setup): symlink .gitconfig instead of copying (6c60d20)
- 2026-03-17: fix(sync): replace git pull with rsync for local installation (73cdc2e)
- 2026-03-16: fix(ci): resolve 3 CI failures from developer tools addition (971e01e)
- 2026-03-12: fix: resolve 26 bugs and close Windows/Linux parity gaps (82481ef)
- 2026-02-28: fix(ci): remove non-ascii characters to resolve PSScriptAnalyzer BOM error (b36fac7)
- 2026-02-26: fix(ci): remove false CLAUDE.md assertion from integration tests (e2cbd36)
- 2026-02-23: fix: close setup parity gaps between Linux and Windows (866314e)
- 2026-02-22: fix: harden AI rules deployment and fix SSH directory copy (0736c74)
- 2026-02-18: fix: replace non-interactive claude plugin install with manual instruction (add9285)
- 2026-02-16: fix: handle zsh nomatch error in secrets_clean glob patterns (791c0e6)
- 2026-02-16: fix: install zsh in CI for zsh compatibility tests (409c347)
- 2026-02-07: fix: harden all shell scripts for POSIX/zsh compatibility and add 95 bats-core tests (196a4e5)
- 2026-01-15: fix: add shellcheck disable for zsh-specific syntax (544a8fe)
- 2026-01-14: fix: remove local keyword, skip GITHUB_ secrets, add CI workflow (6886c3b)
- 2025-03-23: fix: issue in .bashrc (4496a06)

## Refactoring

- 2026-05-15: refactor(claude-md): compact MCP Server Usage Rules to bullets + links (f96afd8)
- 2026-05-15: refactor(claude-md): trim Neural Hive Loop phases + Vault Structure (e6b4b8e)
- 2026-05-15: refactor(claude-md): replace duplicated standards with vault pointers (b7422b0)
- 2026-02-28: refactor(ai): align project init and agent prompts with Neural Hive protocol (2e1eda8)
- 2025-11-28: refactor: standardize shell configs and fix APPS_HOME path (ed02dc3)

## Documentation

- 2026-03-12: docs(readme): update test count, add aider aliases and new scripts (6e687b1)
- 2026-01-12: docs: add SECRETS.md and reorganize documentation structure (c84292e)

## Tests

- 2026-02-28: test(ci): pass PSScriptAnalyzer settings to bats test (b4b178c)

## Chores

- 2026-05-17: chore(TERM-001): scaffold + filled proposal for Ghostty Linux bootstrap (11270f3)
- 2026-05-17: chore(AI-011): archive spec + correct opencode.jsonc Layer 1 comment (f9d2497)
- 2026-05-16: chore(AI-011): full aider sunset — Linux, Windows, README, env-contract (1a6a15f)
- 2025-12-05: chore: add sops age key file path as env variable (f4a58a7)
- 2025-11-19: chore: Remove claude boost.sh script and its installation command from install.sh. (7f96a05)
- 2025-11-14: chore: prioritize ~/go/bin in $PATH to use correct go-task v3 from v2 (ad8a9a8)
- 2025-08-29: chore: add zoho codes (bdc088a)
- 2025-08-29: chore: add cloudflare token (9ac5a47)
- 2025-08-26: chore: add pre-commit hooks and validation script (117dbc9)
- 2025-08-26: chore: add sensitive scripts and refactor documentation (141b1f3)
- 2025-06-20: chore: add age encryption for secrets (3f7ba2d)
- 2025-03-23: chore: add gh secrets configuration from env file (12391d1)
- 2025-03-23: chore: add node dependencies checkout (e91d2f2)
- 2025-03-23: chore: add dependencies checkout for aliases (79dacc2)
- 2025-03-22: chore: add custom functions to zsh terminal (32de27b)

## Other

- 2026-03-01: secrets: add openrouter api key (0df59eb)
- 2026-02-22: doc: migrate docs/ to private vault and extract ADRs (5a045cb)
- 2026-02-21: doc: minor details (01abc2b)
- 2026-02-18: doc: remove excalidraw MCP server from all configs and docs (babdcb1)
- 2026-02-10: git commit -m "feat: add drawio MCP server registration to setup scripts" (7340e37)
- 2026-02-08: doc: update LLM files with obsidian vault info (d002719)
- 2026-02-07: doc: add backlog file (4e88cd0)
- 2026-02-04: doc: add claude code plugins installation (81e88dd)
- 2025-03-24: bug: remove --icon flag from eza aliases (250bd8c)
- 2025-03-23: bug: fix issue in nvm alias inizialiation (fc73945)
- 2024-11-01: First commit (c40614f)
- 2024-11-01: Initial commit (3cb97d3)

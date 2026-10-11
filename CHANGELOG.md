# Changelog

## [0.67.0](https://github.com/mlorentedev/dotfiles/compare/v0.66.0...v0.67.0) (2026-10-10)


### Features

* **contract:** give darwin its own env-contract key ([#2284](https://github.com/mlorentedev/dotfiles/issues/2284)) ([784e034](https://github.com/mlorentedev/dotfiles/commit/784e034b48a0729f134ac10024e4167d7b0bac50))
* **converge:** bind harness hooks as a converge step and report drift in doctor ([#2288](https://github.com/mlorentedev/dotfiles/issues/2288)) ([435689a](https://github.com/mlorentedev/dotfiles/commit/435689a9e08af6665c110b448894c82cb5c5ebcd))
* **converge:** clone or fast-forward the checkout as the first step ([#2263](https://github.com/mlorentedev/dotfiles/issues/2263)) ([d52ef72](https://github.com/mlorentedev/dotfiles/commit/d52ef72193f8770f9009b4c3b62ad3ef8ba884f6))
* **converge:** deploy the ai/deploy.json configs as a converge step ([#2245](https://github.com/mlorentedev/dotfiles/issues/2245)) ([7ab421e](https://github.com/mlorentedev/dotfiles/commit/7ab421e91ea1e1aa8009d1da0f9b9708a0d645ad))
* **converge:** give macOS GUI apps the contract variables through launchd ([#2256](https://github.com/mlorentedev/dotfiles/issues/2256)) ([0433a42](https://github.com/mlorentedev/dotfiles/commit/0433a42f89e145e71ce548e7a01d41507c20198b))
* **converge:** install the packages.json catalog in the tools step on every OS ([#2289](https://github.com/mlorentedev/dotfiles/issues/2289)) ([2cd821d](https://github.com/mlorentedev/dotfiles/commit/2cd821d921fe26464159d5bc4ff6023f3de47013))
* **converge:** run the setup script last and route dotf update through converge ([#2274](https://github.com/mlorentedev/dotfiles/issues/2274)) ([d614102](https://github.com/mlorentedev/dotfiles/commit/d614102a1d43490d67f1c9c5e40f3f0096486b71))
* **doctor:** keep Colima's VM at the declared 4 CPU and 8 GiB ([#2241](https://github.com/mlorentedev/dotfiles/issues/2241)) ([98c78a4](https://github.com/mlorentedev/dotfiles/commit/98c78a494d44e4e51be401fb9aaeb289883fc640))
* **pr-land:** require a declared standing merge grant on the base branch ([#2229](https://github.com/mlorentedev/dotfiles/issues/2229)) ([3e1b993](https://github.com/mlorentedev/dotfiles/commit/3e1b9933c5642f025016ba5ed5479c53419d3bff)), closes [#2178](https://github.com/mlorentedev/dotfiles/issues/2178)
* **secrets:** push a rotated value to its CI consumers with rotate --push-ci ([#2300](https://github.com/mlorentedev/dotfiles/issues/2300)) ([635a99b](https://github.com/mlorentedev/dotfiles/commit/635a99b79309cd73395531a7990855836390d5b4))
* **tools:** pin delta, fd and bat through mise and declare btop ([#2240](https://github.com/mlorentedev/dotfiles/issues/2240)) ([b7afc9b](https://github.com/mlorentedev/dotfiles/commit/b7afc9b6ed6e37a7857d734ce773ea4c6dff2e7b)), closes [#2013](https://github.com/mlorentedev/dotfiles/issues/2013)
* **tools:** pin Python 3.13.16 and PyYAML through mise ([#2222](https://github.com/mlorentedev/dotfiles/issues/2222)) ([e04219c](https://github.com/mlorentedev/dotfiles/commit/e04219c29791b1f1bc3b6a2ba3d85a754be2908d))
* **tools:** pin the daily Kubernetes and lint CLIs through mise ([#2239](https://github.com/mlorentedev/dotfiles/issues/2239)) ([01379f5](https://github.com/mlorentedev/dotfiles/commit/01379f5924dc7f23064cd48b18ef34057ec201b5)), closes [#2013](https://github.com/mlorentedev/dotfiles/issues/2013)
* **tools:** pin the kubelab essentials and probe CLIs without --version ([#2238](https://github.com/mlorentedev/dotfiles/issues/2238)) ([f6557b4](https://github.com/mlorentedev/dotfiles/commit/f6557b46dfae00c5d034e3443e165207b0f57af4)), closes [#2013](https://github.com/mlorentedev/dotfiles/issues/2013)
* **tools:** provision Docker on macOS through Colima ([#2242](https://github.com/mlorentedev/dotfiles/issues/2242)) ([93b4cfc](https://github.com/mlorentedev/dotfiles/commit/93b4cfc9d6ffd56df0bc2ab316beff43784e9d43))
* **tools:** read Homebrew casks as their own system key ([#2234](https://github.com/mlorentedev/dotfiles/issues/2234)) ([040c1fc](https://github.com/mlorentedev/dotfiles/commit/040c1fccd6d5d2f74ba04464645a13a077ca7d28))
* **worktree:** probe process working directories on macOS for sweep and done ([#2293](https://github.com/mlorentedev/dotfiles/issues/2293)) ([95f9906](https://github.com/mlorentedev/dotfiles/commit/95f99060255ad647298ef2ed633611648fbb3db9))


### Bug Fixes

* **agy:** pin Gemini 3.8 Flash and keep the baseline grants ([#2236](https://github.com/mlorentedev/dotfiles/issues/2236)) ([7b7d013](https://github.com/mlorentedev/dotfiles/commit/7b7d013fd1c977b3362a0647f686884a900dbe6a))
* **config:** copy to each OS's clipboard from tmux and complete the terraform on PATH ([#2290](https://github.com/mlorentedev/dotfiles/issues/2290)) ([6052c66](https://github.com/mlorentedev/dotfiles/commit/6052c6653d5c16735f9ec335219b80209a0f3e0d))
* **converge:** mirror the deploy-dir set so macOS can clear repo drift ([#2230](https://github.com/mlorentedev/dotfiles/issues/2230)) ([873ae3a](https://github.com/mlorentedev/dotfiles/commit/873ae3a7a10679d8515b4629e6c595c6486ad7a1))
* **deploy:** merge opencode's tui.json so Orca's plugin key survives a deploy ([#2265](https://github.com/mlorentedev/dotfiles/issues/2265)) ([d11a33b](https://github.com/mlorentedev/dotfiles/commit/d11a33b64fab23aa9e6f976067fe474d037e96d1))
* **deploy:** retire Claude marketplaces through the CLI's registry, not a setup block ([#2259](https://github.com/mlorentedev/dotfiles/issues/2259)) ([b9c3b1d](https://github.com/mlorentedev/dotfiles/commit/b9c3b1da457faa34a5f4c617c3de8236af7b98fe))
* **doctor:** fail when the hive daemon behind hive client does not answer ([#2255](https://github.com/mlorentedev/dotfiles/issues/2255)) ([cbf9e04](https://github.com/mlorentedev/dotfiles/commit/cbf9e04362caf76218ced17edc58b698d7566a9d))
* **harness:** identify bound hooks by command signature when the marker is gone ([#2276](https://github.com/mlorentedev/dotfiles/issues/2276)) ([c4fbea7](https://github.com/mlorentedev/dotfiles/commit/c4fbea71af7011b2075f87e08e126b7156fcd8c2))
* **harness:** keep the standing-grant pointer inside full-only so compact payloads stay strict ([#2248](https://github.com/mlorentedev/dotfiles/issues/2248)) ([ab3edc2](https://github.com/mlorentedev/dotfiles/commit/ab3edc280a1988a28ec850515b678300a0ffb218))
* **harness:** mirror the checkout minus what git ignores ([#2277](https://github.com/mlorentedev/dotfiles/issues/2277)) ([7fc2858](https://github.com/mlorentedev/dotfiles/commit/7fc28586a6fb438350278b9d656a1d25c66806ad))
* **harness:** prune deploy-dir copies the checkout deleted ([#2267](https://github.com/mlorentedev/dotfiles/issues/2267)) ([0fffda7](https://github.com/mlorentedev/dotfiles/commit/0fffda75c84a5cffcdc6997671977d0e6c711df9))
* **orca:** find Orca's data and running instance by Electron's conventions on every OS ([#2295](https://github.com/mlorentedev/dotfiles/issues/2295)) ([722fd69](https://github.com/mlorentedev/dotfiles/commit/722fd69d9996fd6f91785c00ff838f9f698320c3))
* **orca:** hold only PowerShell hooks to the timeout floor ([#2286](https://github.com/mlorentedev/dotfiles/issues/2286)) ([06d5321](https://github.com/mlorentedev/dotfiles/commit/06d5321159d24ce99620a1c847409b368d3f5e68))
* **pi:** realign OpenRouter DeepSeek limits with the catalog and correct PI-002's record ([#2297](https://github.com/mlorentedev/dotfiles/issues/2297)) ([8f4c0ba](https://github.com/mlorentedev/dotfiles/commit/8f4c0babece81966fc6912fe79531c8c6e864c49))
* **secrets:** close CLI-037 with its archive review gaps applied ([#2307](https://github.com/mlorentedev/dotfiles/issues/2307)) ([6d63256](https://github.com/mlorentedev/dotfiles/commit/6d63256710f8ebfd1a8cb33dfb7243284ba633c5))
* **shell:** keep the inherited PATH in .bashrc and export only tool homes that exist ([#2283](https://github.com/mlorentedev/dotfiles/issues/2283)) ([4001e40](https://github.com/mlorentedev/dotfiles/commit/4001e402f8cc58622bf511eef0333fa7bcd58769))
* **tools:** install npm catalog tools under ~/.local and let uv replace foreign entry points ([#2254](https://github.com/mlorentedev/dotfiles/issues/2254)) ([f50b3da](https://github.com/mlorentedev/dotfiles/commit/f50b3da9181dc9583535af580cc9391768e116bd))
* **vault:** leave 50_work product and client records out of dead-ends ([#2233](https://github.com/mlorentedev/dotfiles/issues/2233)) ([bed5b11](https://github.com/mlorentedev/dotfiles/commit/bed5b1158e12422eb629b827f911d8e66647bdc7)), closes [#2197](https://github.com/mlorentedev/dotfiles/issues/2197)
* **vault:** leave linkless genres and attachments out of the link-graph counts ([#2226](https://github.com/mlorentedev/dotfiles/issues/2226)) ([7568dff](https://github.com/mlorentedev/dotfiles/commit/7568dffbe382192932779100c80a1fa782624b9b))
* **vault:** notify through Notification Center on macOS ([#2292](https://github.com/mlorentedev/dotfiles/issues/2292)) ([fd0d059](https://github.com/mlorentedev/dotfiles/commit/fd0d0593744dc25919677a69b55dc623b4d992ee))

## [0.66.0](https://github.com/mlorentedev/dotfiles/compare/v0.65.0...v0.66.0) (2026-10-09)


### Features

* **ci:** draw PR-Agent's reviewer from a NaN + Claude Haiku pool ([#2188](https://github.com/mlorentedev/dotfiles/issues/2188)) ([5dbb724](https://github.com/mlorentedev/dotfiles/commit/5dbb724a19337779a2e57df80c4d07e538d2899f))
* **deploy:** merge a TOML config the tool also writes ([#2192](https://github.com/mlorentedev/dotfiles/issues/2192)) ([e0be824](https://github.com/mlorentedev/dotfiles/commit/e0be824231b9504f94a3a71396dd4b2880adc301))
* **doctor:** report and fix the ~/.local/bin copies that shadow mise's pinned CLIs ([#2196](https://github.com/mlorentedev/dotfiles/issues/2196)) ([90549a9](https://github.com/mlorentedev/dotfiles/commit/90549a9e6250c6e4df81512a35a9f32cdf8cb367))
* **git:** converge git's global config without owning ~/.gitconfig ([#2208](https://github.com/mlorentedev/dotfiles/issues/2208)) ([0ecddb3](https://github.com/mlorentedev/dotfiles/commit/0ecddb362e78b57879602c11c08607dca731481e))
* **lessons:** check lessons with dotf in pre-commit and retire the shell twin ([#2211](https://github.com/mlorentedev/dotfiles/issues/2211)) ([11c56bc](https://github.com/mlorentedev/dotfiles/commit/11c56bc095584a6df539ce8c232fde5e3a9d93cb))
* **setup:** install uv through mise and poetry from the catalog ([#2186](https://github.com/mlorentedev/dotfiles/issues/2186)) ([9bde46c](https://github.com/mlorentedev/dotfiles/commit/9bde46c7b06d01e475f6e9b8ee6b9a053ecf4695))
* **tools:** declare the class-3 CLIs as system entries in the catalog ([#2210](https://github.com/mlorentedev/dotfiles/issues/2210)) ([64a279c](https://github.com/mlorentedev/dotfiles/commit/64a279c28d2cc50167ba4cbb5b2471b3eae68105))
* **tools:** install herdr through mise at 0.9.3 ([#2190](https://github.com/mlorentedev/dotfiles/issues/2190)) ([b9e67ff](https://github.com/mlorentedev/dotfiles/commit/b9e67ff53cddaa61bf12bcd592e5833dd97bff1a))
* **tools:** install mise from the catalog ([#2185](https://github.com/mlorentedev/dotfiles/issues/2185)) ([75869c0](https://github.com/mlorentedev/dotfiles/commit/75869c0098e565ae080b362a991509dafee5b38c))
* **tools:** install Python packages through mise and require Python 3.11 in doctor ([#2219](https://github.com/mlorentedev/dotfiles/issues/2219)) ([41ee37d](https://github.com/mlorentedev/dotfiles/commit/41ee37d8a7b9677745b12327634726d3a1b776f6))


### Bug Fixes

* **deploy:** narrow an existing directory that holds a private file ([#2171](https://github.com/mlorentedev/dotfiles/issues/2171)) ([9a6ab16](https://github.com/mlorentedev/dotfiles/commit/9a6ab16283e8f5081fc04daca82a4743f2a28c8e))
* **doctor:** hold the age root to its declared mode, as secrets verify does ([#2206](https://github.com/mlorentedev/dotfiles/issues/2206)) ([70c4ffa](https://github.com/mlorentedev/dotfiles/commit/70c4ffaa063200e389459bec7cb2432492817d00))
* **harness:** refresh the records only from a vault level with its upstream ([#2175](https://github.com/mlorentedev/dotfiles/issues/2175)) ([84c6c2e](https://github.com/mlorentedev/dotfiles/commit/84c6c2ea6b244ecaba31de9ca0cd148e5769bd5b))
* **harness:** refresh the test skill record's sha and guard record shas in CI ([#2163](https://github.com/mlorentedev/dotfiles/issues/2163)) ([9199066](https://github.com/mlorentedev/dotfiles/commit/9199066bd2321e862e3b2bf975a4b026444f325a))
* **hooks:** fail closed when a declared pre-commit gate cannot run ([#2184](https://github.com/mlorentedev/dotfiles/issues/2184)) ([035068f](https://github.com/mlorentedev/dotfiles/commit/035068f432d792da30b290c314eb0ec53e3bf59e))
* **pi:** declare adaptive thinking on the Anthropic review models ([#2218](https://github.com/mlorentedev/dotfiles/issues/2218)) ([c8ee51d](https://github.com/mlorentedev/dotfiles/commit/c8ee51d7b6e7c5377dd6d09ca02ee1f840d2bd96))
* **pr:** keep an exempt release diff out of the triage queue ([#2209](https://github.com/mlorentedev/dotfiles/issues/2209)) ([9720505](https://github.com/mlorentedev/dotfiles/commit/9720505a611b64f70560bf7aa9b36b1b77947f0c))
* **setup-windows:** report a winget tool installed only when it resolves ([#2159](https://github.com/mlorentedev/dotfiles/issues/2159)) ([7af4667](https://github.com/mlorentedev/dotfiles/commit/7af4667a82ff7f3d0a371875a0c7fca61d32c0c0))
* **setup:** run the linux-amd64 downloads on linux-amd64 only, and remove their leftovers elsewhere ([#2166](https://github.com/mlorentedev/dotfiles/issues/2166)) ([9c25aee](https://github.com/mlorentedev/dotfiles/commit/9c25aee932b75d6b2024e7101a743f66d309c26c))
* **shell:** stop binding zsh-tied variable names in dual-shell scripts ([#2169](https://github.com/mlorentedev/dotfiles/issues/2169)) ([e861c29](https://github.com/mlorentedev/dotfiles/commit/e861c29fbcbb14c208597822e4429dc2deaddbe0))
* **test:** feed the tied-name guard its file on stdin, since BSD sed fails on `--` ([#2181](https://github.com/mlorentedev/dotfiles/issues/2181)) ([0f8add9](https://github.com/mlorentedev/dotfiles/commit/0f8add92a3a9c47053379edfc41c0a4bf4eba6e0))
* **tools:** read a tool's version from stdout before stderr ([#2213](https://github.com/mlorentedev/dotfiles/issues/2213)) ([385f36b](https://github.com/mlorentedev/dotfiles/commit/385f36b130e05c2259c055936e05c73da4e1f897))
* **tools:** turn off mise's self-update notice in the rendered config ([#2198](https://github.com/mlorentedev/dotfiles/issues/2198)) ([44db9fc](https://github.com/mlorentedev/dotfiles/commit/44db9fc53694757d1951541afbef6f97a9a355d9))
* **vault:** stop vault health passing the connection check on an obsidian error ([#2182](https://github.com/mlorentedev/dotfiles/issues/2182)) ([d98adab](https://github.com/mlorentedev/dotfiles/commit/d98adabbf5570b7e7425a2b6866a482f997e3d86))
* **windows:** reconcile pi packages without dotf on PATH ([#2221](https://github.com/mlorentedev/dotfiles/issues/2221)) ([981d37e](https://github.com/mlorentedev/dotfiles/commit/981d37e0e69b36788ea5c918615023d51ba07d8d))

## [0.65.0](https://github.com/mlorentedev/dotfiles/compare/v0.64.0...v0.65.0) (2026-10-08)


### Features

* **converge:** add dotf converge over an ordered reconciler registry ([#2020](https://github.com/mlorentedev/dotfiles/issues/2020)) ([fe5174a](https://github.com/mlorentedev/dotfiles/commit/fe5174a1c01fe42e6a6cf38e7075d58fd0d8a855))
* **converge:** deploy agent instructions first with a records-harness reconciler ([#2025](https://github.com/mlorentedev/dotfiles/issues/2025)) ([5f411e9](https://github.com/mlorentedev/dotfiles/commit/5f411e91c186851eaecc0248304cddcf8316ef40))
* **converge:** install the pinned CLIs as the tools step of dotf converge ([#2041](https://github.com/mlorentedev/dotfiles/issues/2041)) ([ea05ea9](https://github.com/mlorentedev/dotfiles/commit/ea05ea9915e92c8313a88b9f1e3b6b684b98b43d))
* **converge:** persist each applied run's report under the state dir ([#2033](https://github.com/mlorentedev/dotfiles/issues/2033)) ([1392964](https://github.com/mlorentedev/dotfiles/commit/1392964c34405d72096dbe8f076c6f8f56ffa962))
* **deploy:** an OS selector on deploy entries, read a release before it is used ([#2049](https://github.com/mlorentedev/dotfiles/issues/2049)) ([946652d](https://github.com/mlorentedev/dotfiles/commit/946652d60229dd7d80c1958f25c84936c10eb7ca))
* **deploy:** deploy the ssh config and public key through dotf deploy on every OS ([#2051](https://github.com/mlorentedev/dotfiles/issues/2051)) ([c712c31](https://github.com/mlorentedev/dotfiles/commit/c712c31268c6bd45000983dde9e03de4ac87ce6e))
* **deploy:** deploy the zsh and tmux rc files through dotf deploy, safely on every OS ([#2043](https://github.com/mlorentedev/dotfiles/issues/2043)) ([b4c44bc](https://github.com/mlorentedev/dotfiles/commit/b4c44bcdaca92272f41d69bc35b3812649ced57d))
* **doctor:** read a darwin machine as darwin, and check the pinned CLIs through mise ([#2048](https://github.com/mlorentedev/dotfiles/issues/2048)) ([5e34a9d](https://github.com/mlorentedev/dotfiles/commit/5e34a9de730c4c113198ef92df8cfd5cb950ed43))
* **harness:** deploy the agents' instruction files from Go with dotf harness instructions ([#2044](https://github.com/mlorentedev/dotfiles/issues/2044)) ([9daf0b4](https://github.com/mlorentedev/dotfiles/commit/9daf0b46062ec0169b0a56d049dd23f2bbb13a6b))
* **lessons:** one lesson format, a generated index, and dotf lessons fmt ([#2039](https://github.com/mlorentedev/dotfiles/issues/2039)) ([1268831](https://github.com/mlorentedev/dotfiles/commit/1268831260665dba5fd0190fc0313e61c73e2029))
* **mem:** run the session brief's vault health in-process ([#2110](https://github.com/mlorentedev/dotfiles/issues/2110)) ([c3b74c4](https://github.com/mlorentedev/dotfiles/commit/c3b74c42d6fe5e68b529a1eac0cb170a97a00387))
* **pr:** land a PR only when CI, triage and freshness hold on one head ([#2037](https://github.com/mlorentedev/dotfiles/issues/2037)) ([517cc59](https://github.com/mlorentedev/dotfiles/commit/517cc597315da322c0ff24bc8367eb175531c7e5))
* **pr:** land several PRs as one queue, one at a time, under a per-repo lock ([#2067](https://github.com/mlorentedev/dotfiles/issues/2067)) ([4c69afe](https://github.com/mlorentedev/dotfiles/commit/4c69afe7ed2f5c708c0b4c0196113cc6a57ecb0a))
* **secrets:** register the escrow entry for ace2's restic password ([#2148](https://github.com/mlorentedev/dotfiles/issues/2148)) ([198832f](https://github.com/mlorentedev/dotfiles/commit/198832f5d5e2d108170e7ec1432ead49bf142313))
* **tools:** install the pinned CLIs through mise with dotf tools sync ([#2030](https://github.com/mlorentedev/dotfiles/issues/2030)) ([a682b6e](https://github.com/mlorentedev/dotfiles/commit/a682b6eb7f5b81fd571e6fc8d2ffa3c99401e666))
* **tools:** let a release asset be keyed by GOOS/GOARCH ([#2029](https://github.com/mlorentedev/dotfiles/issues/2029)) ([9fed8cf](https://github.com/mlorentedev/dotfiles/commit/9fed8cf8f74b3a5357f69e753acc342fedd749ae))
* **tools:** read source.type system and install through the OS package manager ([#2060](https://github.com/mlorentedev/dotfiles/issues/2060)) ([93b6c19](https://github.com/mlorentedev/dotfiles/commit/93b6c19d836b5a827737db434b9262410da37929))
* **tools:** refuse to install a binary that does not run here ([#2014](https://github.com/mlorentedev/dotfiles/issues/2014)) ([9db6df4](https://github.com/mlorentedev/dotfiles/commit/9db6df452ea89fe11a264a510894346743cddce3))


### Bug Fixes

* **agy:** deploy AGY.md and .geminiignore through dotf deploy on every OS ([#2042](https://github.com/mlorentedev/dotfiles/issues/2042)) ([be92ad8](https://github.com/mlorentedev/dotfiles/commit/be92ad85e419ca3ca6c3456bd45054b30e93102e))
* **ci:** judge a single workflow file the same as several in the action-pin check ([#2133](https://github.com/mlorentedev/dotfiles/issues/2133)) ([d27014b](https://github.com/mlorentedev/dotfiles/commit/d27014bcb68075004544e1d900add2120e3a1369))
* **ci:** require the changes job that every filtered required job needs ([#2108](https://github.com/mlorentedev/dotfiles/issues/2108)) ([a23384c](https://github.com/mlorentedev/dotfiles/commit/a23384cb56385d90c95a9f03d7a2814fa26612b5))
* **cli:** keep gh's stderr in the error when a gh call fails ([#2088](https://github.com/mlorentedev/dotfiles/issues/2088)) ([7f8a974](https://github.com/mlorentedev/dotfiles/commit/7f8a9741db3da2932c892a5069dd3d7205c673ed))
* **cli:** refuse an unknown subcommand under a command group ([#2093](https://github.com/mlorentedev/dotfiles/issues/2093)) ([24283b3](https://github.com/mlorentedev/dotfiles/commit/24283b34038a595966a5640d21e6204fc38da2df))
* **cli:** report a mistyped flag or refused arguments under SilenceErrors ([#2092](https://github.com/mlorentedev/dotfiles/issues/2092)) ([9e028c9](https://github.com/mlorentedev/dotfiles/commit/9e028c9c196118263fcd2cfe5bb3fab2e070d8b1))
* **deploy:** converge a file's mode only toward narrower, and let doctor see it ([#2111](https://github.com/mlorentedev/dotfiles/issues/2111)) ([a53dcd6](https://github.com/mlorentedev/dotfiles/commit/a53dcd66691b963d6fdeb7f3b52e05c182bcf111)), closes [#1664](https://github.com/mlorentedev/dotfiles/issues/1664)
* **deploy:** derive a created directory's mode from every entry sharing it ([#2098](https://github.com/mlorentedev/dotfiles/issues/2098)) ([bc1ecf2](https://github.com/mlorentedev/dotfiles/commit/bc1ecf287476ad580306071161cbb98c260547a8))
* **deploy:** put every deploy report line's columns in one place ([#2112](https://github.com/mlorentedev/dotfiles/issues/2112)) ([6374fe4](https://github.com/mlorentedev/dotfiles/commit/6374fe45df53bb04eedde819344c5821a01155d5)), closes [#1664](https://github.com/mlorentedev/dotfiles/issues/1664)
* **deploy:** treat a symlinked destination as drift and keep the link as its backup ([#2104](https://github.com/mlorentedev/dotfiles/issues/2104)) ([4086379](https://github.com/mlorentedev/dotfiles/commit/408637973ff38d13c4702fc0111c2ec900e45e29))
* **doctor:** compare directories as files, not by OS-folded spelling ([#2099](https://github.com/mlorentedev/dotfiles/issues/2099)) ([a3e75cf](https://github.com/mlorentedev/dotfiles/commit/a3e75cf65731bbafee77f3e5c3304ecf5a15505f)), closes [#2094](https://github.com/mlorentedev/dotfiles/issues/2094)
* **doctor:** decide the target OS through the GOOS seam only ([#2095](https://github.com/mlorentedev/dotfiles/issues/2095)) ([7f648fd](https://github.com/mlorentedev/dotfiles/commit/7f648fd60c1fd24c4246981ed0453fb04907eefb))
* **doctor:** name the remedy when agy's master MCP config is empty or invalid ([#2143](https://github.com/mlorentedev/dotfiles/issues/2143)) ([12978cc](https://github.com/mlorentedev/dotfiles/commit/12978cc3761f17c2e8d244e6f78a162ee7168b62))
* **doctor:** warn when an applicable rendered deploy entry was never deployed ([#2103](https://github.com/mlorentedev/dotfiles/issues/2103)) ([03f1e51](https://github.com/mlorentedev/dotfiles/commit/03f1e51d4a5b582d11e1afa6ae138d42ccaf6c61)), closes [#2100](https://github.com/mlorentedev/dotfiles/issues/2100)
* **harness:** run compile-harness on macOS bash 3.2 ([#2015](https://github.com/mlorentedev/dotfiles/issues/2015)) ([c9b8767](https://github.com/mlorentedev/dotfiles/commit/c9b8767e16507d65bb8a044222e4eec34b0667f8)), closes [#2013](https://github.com/mlorentedev/dotfiles/issues/2013)
* **harness:** treat a refreshed source region as instruction-file drift ([#2040](https://github.com/mlorentedev/dotfiles/issues/2040)) ([bb4e220](https://github.com/mlorentedev/dotfiles/commit/bb4e220f9d1a77bcdff89793e2c68aa0bb8a4c55))
* **init:** scaffold CI for the repository's forge and default branch ([#2116](https://github.com/mlorentedev/dotfiles/issues/2116)) ([7b42ea6](https://github.com/mlorentedev/dotfiles/commit/7b42ea6ed0dbd3160db749d9a7ec340fdfa471a8))
* **mem:** archive only the session's own thread in the fallback journal ([#2077](https://github.com/mlorentedev/dotfiles/issues/2077)) ([47c5e22](https://github.com/mlorentedev/dotfiles/commit/47c5e22d5dc315f800880e291b1b5c30ce911848))
* **mem:** fail the thread key when the working directory Getwd returned no longer exists ([#2086](https://github.com/mlorentedev/dotfiles/issues/2086)) ([ed7e1fa](https://github.com/mlorentedev/dotfiles/commit/ed7e1fad08e878c6818c72eb8afae0e032013092))
* **memlink:** link auto-memory when the repo name differs from the vault slug ([#2026](https://github.com/mlorentedev/dotfiles/issues/2026)) ([2cff59f](https://github.com/mlorentedev/dotfiles/commit/2cff59f495da561aac43f1b9956a4ed0c5ac1f85))
* **mem:** name the session-end fallback journal for the agent that ends it ([#2105](https://github.com/mlorentedev/dotfiles/issues/2105)) ([042409f](https://github.com/mlorentedev/dotfiles/commit/042409f073c1e9b224c159858c4d7de49e21bc9e))
* **mem:** refuse a handoff body line the parser reads as structure ([#2075](https://github.com/mlorentedev/dotfiles/issues/2075)) ([f212c89](https://github.com/mlorentedev/dotfiles/commit/f212c89513f919dc87dd7a4b1f29f07bc8e41ce6))
* **mem:** resolve a relative gitdir pointer against its file, and fail on an unreadable cwd ([#2076](https://github.com/mlorentedev/dotfiles/issues/2076)) ([3800daf](https://github.com/mlorentedev/dotfiles/commit/3800daff515b97f1250bd59ab6066957d5fcf928))
* **mem:** sanitise journal names and name a fork's journal ([#2035](https://github.com/mlorentedev/dotfiles/issues/2035)) ([a216b23](https://github.com/mlorentedev/dotfiles/commit/a216b238198f218a7c2aedd59e10b2444d9466a9))
* **mem:** treat the default branch of any remote as ambient in the thread key ([#2101](https://github.com/mlorentedev/dotfiles/issues/2101)) ([ee09a5c](https://github.com/mlorentedev/dotfiles/commit/ee09a5c66f9f258c26ca2d64c5faba2dae4a9fd4))
* **mem:** treat the remote's default branch as ambient in the thread key ([#2080](https://github.com/mlorentedev/dotfiles/issues/2080)) ([40c32a1](https://github.com/mlorentedev/dotfiles/commit/40c32a1960c3fdb4042d7c663fa84496d4e8f309))
* **opencode:** drop the two OpenRouter free models the catalog retired ([#2074](https://github.com/mlorentedev/dotfiles/issues/2074)) ([580ab10](https://github.com/mlorentedev/dotfiles/commit/580ab10f23f840c68fcd2f9fe03f7c607711c85d))
* **orca:** unique temp file for tuned hooks, tuner under the complexity limit ([#2140](https://github.com/mlorentedev/dotfiles/issues/2140)) ([cd36e58](https://github.com/mlorentedev/dotfiles/commit/cd36e587d0fd1d18b96ad9d02cbb703f8cf36247))
* **pi:** align OpenRouter DeepSeek limits ([#2011](https://github.com/mlorentedev/dotfiles/issues/2011)) ([946afa5](https://github.com/mlorentedev/dotfiles/commit/946afa5002268537fa983709116e30f061d986c5))
* **pi:** refuse a live package entry the reconcile cannot read ([#2127](https://github.com/mlorentedev/dotfiles/issues/2127)) ([a6e46e3](https://github.com/mlorentedev/dotfiles/commit/a6e46e3bad91bff54f60eab19d63ea3524f5a0ac))
* **pr-agent:** report a GitHub API failure in the publish guard as an outage ([#2071](https://github.com/mlorentedev/dotfiles/issues/2071)) ([2ebba7f](https://github.com/mlorentedev/dotfiles/commit/2ebba7ff117014eaa0116c3b0d608cb435e64c6c))
* **pr:** give an uncomputed merge state its own wait budget in pr land ([#2123](https://github.com/mlorentedev/dotfiles/issues/2123)) ([f51f4f7](https://github.com/mlorentedev/dotfiles/commit/f51f4f7d048f850d9c1398be981937eb1014a020))
* **pr:** say that pr land updated the branch when it still refuses as BEHIND ([#2141](https://github.com/mlorentedev/dotfiles/issues/2141)) ([5f635c2](https://github.com/mlorentedev/dotfiles/commit/5f635c2d12d49c623bb605a6cdd495960978544b))
* **pr:** treat a gh pr checks failure other than no checks as an error ([#2137](https://github.com/mlorentedev/dotfiles/issues/2137)) ([89759d5](https://github.com/mlorentedev/dotfiles/commit/89759d5a2572483355e4bf54b27f3c2f27177fc3))
* **pr:** update a PR again when the base moves while its new CI runs ([#2045](https://github.com/mlorentedev/dotfiles/issues/2045)) ([f2f165b](https://github.com/mlorentedev/dotfiles/commit/f2f165b7377ebd66b34e662d5889d652600f2103))
* **secrets:** redact a prefix left at end of stream only when it is material ([#2128](https://github.com/mlorentedev/dotfiles/issues/2128)) ([1767c8d](https://github.com/mlorentedev/dotfiles/commit/1767c8d611ee635a93135b6af38f4d76ba05d517)), closes [#1857](https://github.com/mlorentedev/dotfiles/issues/1857)
* **shell:** make scripts, hooks and tests work on BSD userland and bash 3.2 ([#2145](https://github.com/mlorentedev/dotfiles/issues/2145)) ([aa2f0f5](https://github.com/mlorentedev/dotfiles/commit/aa2f0f5dd00a0a090391b38d86440abf31fcbf2a))
* **spec-gate:** read closing keywords from the branch commits as well as the body ([#2114](https://github.com/mlorentedev/dotfiles/issues/2114)) ([a2a8bbc](https://github.com/mlorentedev/dotfiles/commit/a2a8bbc1099736eca87e4050cb7129a3826cc550)), closes [#1878](https://github.com/mlorentedev/dotfiles/issues/1878)
* **spec-gate:** resolve the open PR of the branch being pushed ([#2113](https://github.com/mlorentedev/dotfiles/issues/2113)) ([5ae785e](https://github.com/mlorentedev/dotfiles/commit/5ae785eb27d481e0d9dd59d41a3e24c5b9206a28))
* **spec:** follow a renamed spec folder back to where its work began ([#2131](https://github.com/mlorentedev/dotfiles/issues/2131)) ([8a610b8](https://github.com/mlorentedev/dotfiles/commit/8a610b8161b7670ddfc4c577dba9af0c7a0372ad))
* **spec:** link FIX-WIN-TUI-001 to the issue that tracks its archive ([#2144](https://github.com/mlorentedev/dotfiles/issues/2144)) ([8544b99](https://github.com/mlorentedev/dotfiles/commit/8544b990af3f8db22879dd972992137a5bed9335)), closes [#2019](https://github.com/mlorentedev/dotfiles/issues/2019)
* **spec:** name a pi turn cap when a foreground review writes no verdict ([#2130](https://github.com/mlorentedev/dotfiles/issues/2130)) ([5bb9ecf](https://github.com/mlorentedev/dotfiles/commit/5bb9ecfecc00a12d637367ea938f34c342b226e7))
* **spec:** name a pi turn cap when archive finds a detached review wrote no verdict ([#2134](https://github.com/mlorentedev/dotfiles/issues/2134)) ([5096a3c](https://github.com/mlorentedev/dotfiles/commit/5096a3c5e8a3dd51c30138ed159ec83d623fefa5))
* **spec:** refuse to archive a review that has no review-request.json ([#2155](https://github.com/mlorentedev/dotfiles/issues/2155)) ([0e7e504](https://github.com/mlorentedev/dotfiles/commit/0e7e504854711dcc9e4c8de584b9844be5aab0c9))
* **spec:** rewrite the archived status in CRLF proposals, and add it where none exists ([#2032](https://github.com/mlorentedev/dotfiles/issues/2032)) ([0723625](https://github.com/mlorentedev/dotfiles/commit/0723625fb55f09e6880e6f14f041d397e1fbe18c))
* **spec:** tick the archive checklist items the archive performed ([#2073](https://github.com/mlorentedev/dotfiles/issues/2073)) ([b0c755b](https://github.com/mlorentedev/dotfiles/commit/b0c755b37dcb756c549ee05e3a01cf6975e00209))
* **spec:** verify the work-gate issue over REST ([#2129](https://github.com/mlorentedev/dotfiles/issues/2129)) ([685cf4e](https://github.com/mlorentedev/dotfiles/commit/685cf4e22a2baaec410e3c3c3ced22901055d6e4))
* **tools:** make the install dry run refuse what install refuses ([#2109](https://github.com/mlorentedev/dotfiles/issues/2109)) ([2a3c743](https://github.com/mlorentedev/dotfiles/commit/2a3c7439d4e83330727c70691766eaeec2bd794b)), closes [#1892](https://github.com/mlorentedev/dotfiles/issues/1892)
* **vault:** find a repository's existing entry by repo_url before creating one ([#2117](https://github.com/mlorentedev/dotfiles/issues/2117)) ([064a01d](https://github.com/mlorentedev/dotfiles/commit/064a01d3f31f295f1fc4883a38bedcd14b8a6e4e))
* **vault:** leave expected orphans and unresolved links out of the health counts ([#2154](https://github.com/mlorentedev/dotfiles/issues/2154)) ([52bc3fc](https://github.com/mlorentedev/dotfiles/commit/52bc3fc231f743f42390d646f71bbeaab3224847))
* **worktree:** let done remove a squash-merged branch whose work landed ([#2107](https://github.com/mlorentedev/dotfiles/issues/2107)) ([28cc209](https://github.com/mlorentedev/dotfiles/commit/28cc20993ccfa9a29f96155f64be8053f7576632))
* **worktree:** let done take the slug add was given ([#2125](https://github.com/mlorentedev/dotfiles/issues/2125)) ([3c821a6](https://github.com/mlorentedev/dotfiles/commit/3c821a68c1d8523d8b2983dcc69c4c0da365c4b9))
* **worktree:** refuse to remove the worktree holding the caller's directory ([#2142](https://github.com/mlorentedev/dotfiles/issues/2142)) ([95b4cb7](https://github.com/mlorentedev/dotfiles/commit/95b4cb71a39f08d8b5d3e034d55582b7f1fd165b))

## [0.64.0](https://github.com/mlorentedev/dotfiles/compare/v0.63.0...v0.64.0) (2026-10-04)


### Features

* **deploy:** declare Claude Code's settings.json as a merge entry ([#1996](https://github.com/mlorentedev/dotfiles/issues/1996)) ([944993e](https://github.com/mlorentedev/dotfiles/commit/944993e258822e620e38e2f06a3ac4ed245fd836))
* **deploy:** install Claude Code plugins from a bare dotf deploy ([#1992](https://github.com/mlorentedev/dotfiles/issues/1992)) ([614ae4e](https://github.com/mlorentedev/dotfiles/commit/614ae4ede087c705cb94c5d7891f7f4b9a1d4747)), closes [#1339](https://github.com/mlorentedev/dotfiles/issues/1339)
* **deploy:** register Claude Code MCP servers from mcp-servers.json ([#1994](https://github.com/mlorentedev/dotfiles/issues/1994)) ([0f67099](https://github.com/mlorentedev/dotfiles/commit/0f67099a4909b22a217467c882695138ce468391))
* **secrets:** declare kubelab's restic passwords as escrow entries ([#1998](https://github.com/mlorentedev/dotfiles/issues/1998)) ([66da55b](https://github.com/mlorentedev/dotfiles/commit/66da55b58a244286869d1e30eb7fddc1e7e7da95))
* **tools:** install hive through a uv-tool catalog source on POSIX ([#2001](https://github.com/mlorentedev/dotfiles/issues/2001)) ([eb14b0d](https://github.com/mlorentedev/dotfiles/commit/eb14b0df8783ecb4b9ddb492739c588423ff951a))


### Bug Fixes

* **ci:** skip PR-Agent when every file is of a type it never reads ([#1986](https://github.com/mlorentedev/dotfiles/issues/1986)) ([b0fcfb7](https://github.com/mlorentedev/dotfiles/commit/b0fcfb7822b6da69d0fb76b6a29556dbf2cea0b5))
* **harness:** narrow fast-track issue exception to &lt;50 LOC ([#1954](https://github.com/mlorentedev/dotfiles/issues/1954)) ([#1972](https://github.com/mlorentedev/dotfiles/issues/1972)) ([e6e894d](https://github.com/mlorentedev/dotfiles/commit/e6e894d05ee4faafca2f229f7e4dcc3009034cad))
* **harness:** window insights handoff decisions by thread date ([#1984](https://github.com/mlorentedev/dotfiles/issues/1984)) ([8bb09c2](https://github.com/mlorentedev/dotfiles/commit/8bb09c2f63f5b83212ceb7904a19492b8f2c8633))
* **windows:** bypass dotf secrets run pipe limitation for TUIs ([#1970](https://github.com/mlorentedev/dotfiles/issues/1970)) ([#1971](https://github.com/mlorentedev/dotfiles/issues/1971)) ([be386de](https://github.com/mlorentedev/dotfiles/commit/be386de1ee930b8a50f6619d9659ce97bf0b76d0))

## [0.63.0](https://github.com/mlorentedev/dotfiles/compare/v0.62.0...v0.63.0) (2026-10-02)


### Features

* **secrets:** let backup acquire its own bw CLI session ([#1958](https://github.com/mlorentedev/dotfiles/issues/1958)) ([378f3f0](https://github.com/mlorentedev/dotfiles/commit/378f3f034b796b7ddcde16c394ea7a5048a789c4))


### Bug Fixes

* **deploy:** re-tune Orca's Copilot hooks on every bare deploy ([#1960](https://github.com/mlorentedev/dotfiles/issues/1960)) ([5c0215c](https://github.com/mlorentedev/dotfiles/commit/5c0215ce4c46e2415225f6835813291b7ecb7a53))
* **doctor:** watch every metered NaN model, not only the bound ones ([#1956](https://github.com/mlorentedev/dotfiles/issues/1956)) ([01133f7](https://github.com/mlorentedev/dotfiles/commit/01133f74c25e2f065fe08f8a4c860dd6ed476a79))
* **harness:** fail on an unclosed full-only region and keep doctrine file modes; archive HARNESS-084 ([#1964](https://github.com/mlorentedev/dotfiles/issues/1964)) ([b366af4](https://github.com/mlorentedev/dotfiles/commit/b366af47dbb7b46dbac73be975d44e46b87e788f))
* **harness:** run the roster drift guard from bats; archive HARNESS-046 ([#1965](https://github.com/mlorentedev/dotfiles/issues/1965)) ([b73267a](https://github.com/mlorentedev/dotfiles/commit/b73267a0fb530aaf299ebdc008bb4fd82d464d34))
* **pi:** declare openrouter deepseek-chat at the provider's limits ([#1952](https://github.com/mlorentedev/dotfiles/issues/1952)) ([883d3ed](https://github.com/mlorentedev/dotfiles/commit/883d3eda6e4dea112949a4e76263ee162b6e8ba0))
* prevent bw serve window popup and fix deepseek limits ([#1969](https://github.com/mlorentedev/dotfiles/issues/1969)) ([6630be9](https://github.com/mlorentedev/dotfiles/commit/6630be91c073972f9336c60e87216ca612520a02))
* resolve pi extension loading conflicts (typebox/mcp) ([#1967](https://github.com/mlorentedev/dotfiles/issues/1967)) ([c8ebfec](https://github.com/mlorentedev/dotfiles/commit/c8ebfec16b5359f76e9c490a0c47f0f2649dcd23))
* **secrets:** wait out bw serve's empty listing during a forced sync ([#1950](https://github.com/mlorentedev/dotfiles/issues/1950)) ([fcc78ee](https://github.com/mlorentedev/dotfiles/commit/fcc78eea2b64b9b0c5b73f0b8550c59e8c2dad6d))

## [0.62.0](https://github.com/mlorentedev/dotfiles/compare/v0.61.0...v0.62.0) (2026-10-02)


### Features

* **harness:** daily canary that probes every bound NaN model ([#1943](https://github.com/mlorentedev/dotfiles/issues/1943)) ([f8c1d76](https://github.com/mlorentedev/dotfiles/commit/f8c1d76e329d0a8980faa1783b353422cd99a45f))
* **harness:** guard every model pin site and archive HARNESS-067 ([#1887](https://github.com/mlorentedev/dotfiles/issues/1887)) ([fe34286](https://github.com/mlorentedev/dotfiles/commit/fe3428610ed86da8727d2a4a6e119f8b4ffbaa81))
* implement Fast-Track spec archiving in CLI ([#1918](https://github.com/mlorentedev/dotfiles/issues/1918)) ([bac2196](https://github.com/mlorentedev/dotfiles/commit/bac2196af150b9d7a7dd47a94b2b1b6132d53e19))
* **pi:** compact at 40% of each model window via native overrides ([#1938](https://github.com/mlorentedev/dotfiles/issues/1938)) ([598f608](https://github.com/mlorentedev/dotfiles/commit/598f608e53e95c9e4eb94e8ca4f630baadfdcc38))
* **secrets:** register Gitea token ([#1846](https://github.com/mlorentedev/dotfiles/issues/1846)) ([cc25a0c](https://github.com/mlorentedev/dotfiles/commit/cc25a0c41970a098897baac0cf23f1ba2e3de07c))
* **secrets:** register the leaving-denver seller passphrase ([#1936](https://github.com/mlorentedev/dotfiles/issues/1936)) ([118ab0f](https://github.com/mlorentedev/dotfiles/commit/118ab0f62a0fe5c5d80caf9341433799ee2d1767))
* **spec:** refuse a new spec while the repository is at its WIP limit ([#1861](https://github.com/mlorentedev/dotfiles/issues/1861)) ([9cfecc8](https://github.com/mlorentedev/dotfiles/commit/9cfecc832715679ea72f82e1d013b0cda8e87f16))
* **tools:** add a dry run and read the catalog from the checkout first ([#1848](https://github.com/mlorentedev/dotfiles/issues/1848)) ([a94ae34](https://github.com/mlorentedev/dotfiles/commit/a94ae3465a245b4a29af25e4b8a056d0f4cd62d0))


### Bug Fixes

* **ai:** align NaN base-plan catalog limits ([#1916](https://github.com/mlorentedev/dotfiles/issues/1916)) ([67ed3b2](https://github.com/mlorentedev/dotfiles/commit/67ed3b2a866499d889af1ab5c34f54fd9cf46a77))
* **ci:** cover every path the suite reads in the code filter ([#1870](https://github.com/mlorentedev/dotfiles/issues/1870)) ([0013ea5](https://github.com/mlorentedev/dotfiles/commit/0013ea5b8b45cec7ab950e91b93eb8f1defc0025))
* **cli:** drop the spec id from the Orca hook output and guard the cleaned files ([#1899](https://github.com/mlorentedev/dotfiles/issues/1899)) ([61db65a](https://github.com/mlorentedev/dotfiles/commit/61db65ac2dc3daa941f0aad4e351987209bc571e))
* **cli:** keep internal ids out of everything dotf prints ([#1917](https://github.com/mlorentedev/dotfiles/issues/1917)) ([bad66a0](https://github.com/mlorentedev/dotfiles/commit/bad66a0158449ae9d13a6cfb2e41e3ffb9f32c6d))
* **cli:** report the module version for go install builds ([#1844](https://github.com/mlorentedev/dotfiles/issues/1844)) ([8a00d89](https://github.com/mlorentedev/dotfiles/commit/8a00d89710e159b00577acde023ec2356bdf0d3c))
* **doctor:** count a PATH directory reached through a symlink once ([#1901](https://github.com/mlorentedev/dotfiles/issues/1901)) ([bf2023c](https://github.com/mlorentedev/dotfiles/commit/bf2023cadafd406a5da709d5f3a59d70ddb21c5f))
* **doctor:** recognize linked worktree checkouts ([#1835](https://github.com/mlorentedev/dotfiles/issues/1835)) ([934e4a7](https://github.com/mlorentedev/dotfiles/commit/934e4a76e13228d9d84ab943ce4bd52b72e3aa3d))
* **doctor:** report a never-written profile as missing, and make the heal test tell its sources apart ([#1863](https://github.com/mlorentedev/dotfiles/issues/1863)) ([1722a9f](https://github.com/mlorentedev/dotfiles/commit/1722a9fdc090190a8e5a14a437c564da7a17848b))
* **doctor:** skip the mapping check when no sync is wired ([#1841](https://github.com/mlorentedev/dotfiles/issues/1841)) ([db2b904](https://github.com/mlorentedev/dotfiles/commit/db2b904794b4700b1e9b5c784d7b570bdc46fae0))
* **doctor:** sync the vault before comparing it with the DR escrow ([#1839](https://github.com/mlorentedev/dotfiles/issues/1839)) ([b0782ad](https://github.com/mlorentedev/dotfiles/commit/b0782adc9fac7b70742c39df71747cd467d04906))
* **doctor:** warn when a package catalog copy exists but cannot be read ([#1902](https://github.com/mlorentedev/dotfiles/issues/1902)) ([8670f87](https://github.com/mlorentedev/dotfiles/commit/8670f87e79537cc0712d4011540c6f9f9436afbf))
* **env:** reserve the ownership marker's name and validate names on every read path ([#1862](https://github.com/mlorentedev/dotfiles/issues/1862)) ([3cb3077](https://github.com/mlorentedev/dotfiles/commit/3cb307748b91412f3ab374002f7b62b9ff8b60c7)), closes [#1363](https://github.com/mlorentedev/dotfiles/issues/1363)
* **harness:** emit executable agy hooks on Windows ([#1827](https://github.com/mlorentedev/dotfiles/issues/1827)) ([5c3102a](https://github.com/mlorentedev/dotfiles/commit/5c3102a5584472a402f791433a88d2843cf7040a))
* **harness:** refuse a manifest target that escapes the checkout ([#1897](https://github.com/mlorentedev/dotfiles/issues/1897)) ([0954368](https://github.com/mlorentedev/dotfiles/commit/09543686fc21fde2b5a864cb29a953f464ccfb3c))
* **harness:** replace read-only Windows mirrors ([#1825](https://github.com/mlorentedev/dotfiles/issues/1825)) ([64eb589](https://github.com/mlorentedev/dotfiles/commit/64eb589d72f5986e81b6819628e7237f90870bc9))
* **harness:** sweep gate journals past a 30-day retention when a new scope opens ([#1945](https://github.com/mlorentedev/dotfiles/issues/1945)) ([5fcaf18](https://github.com/mlorentedev/dotfiles/commit/5fcaf18f0f2c9ca7bb3e3d454c6b51b5da195118))
* **harness:** the doctrine cap warnings name both units, and HARNESS-111's AC3 records the fold [#1685](https://github.com/mlorentedev/dotfiles/issues/1685) shipped ([#1868](https://github.com/mlorentedev/dotfiles/issues/1868)) ([52a09ce](https://github.com/mlorentedev/dotfiles/commit/52a09cebe582b7b23f15bdf4fc12e4669f3bcfb5)), closes [#1241](https://github.com/mlorentedev/dotfiles/issues/1241)
* **mem:** lock handoff-write against lost threads and archive HARNESS-088 ([#1886](https://github.com/mlorentedev/dotfiles/issues/1886)) ([512cd05](https://github.com/mlorentedev/dotfiles/commit/512cd05751bf7dfd626fb2432ae55602407ad264))
* **mem:** lock session-end handoff reads ([#1933](https://github.com/mlorentedev/dotfiles/issues/1933)) ([e5541c6](https://github.com/mlorentedev/dotfiles/commit/e5541c6a304a8350edda16efa90a2adea0859af3))
* **mem:** refuse a handoff thread key that contains whitespace ([#1896](https://github.com/mlorentedev/dotfiles/issues/1896)) ([1cfceaa](https://github.com/mlorentedev/dotfiles/commit/1cfceaa07ab360fd1f3ed7a4effd255719fc7507))
* **pi:** align OpenRouter DeepSeek limits ([#1831](https://github.com/mlorentedev/dotfiles/issues/1831)) ([e783e47](https://github.com/mlorentedev/dotfiles/commit/e783e47699bb6741e3bb1cac659496f190fa3e0f))
* **pr-agent:** bound the primary's attempt and retry once on the fallback ([#1914](https://github.com/mlorentedev/dotfiles/issues/1914)) ([ac9e186](https://github.com/mlorentedev/dotfiles/commit/ac9e186a273abdcb7f5169bafac7f9d13001eca5))
* **pr-agent:** count rebased pushes and ignore quoted review state in the push gate ([#1895](https://github.com/mlorentedev/dotfiles/issues/1895)) ([aa1c497](https://github.com/mlorentedev/dotfiles/commit/aa1c497903d6fdc1d0660464ddb83d10925cf917))
* **pr-agent:** retire mimo-v2.5 and skip a dead model before the review ([#1856](https://github.com/mlorentedev/dotfiles/issues/1856)) ([0a584d4](https://github.com/mlorentedev/dotfiles/commit/0a584d4ebf173fec0e02d77d70134b7712a3bb3b))
* **pr-agent:** stream every NaN call so a held review cannot sit silent ([#1939](https://github.com/mlorentedev/dotfiles/issues/1939)) ([ebe2c05](https://github.com/mlorentedev/dotfiles/commit/ebe2c05458b07b3ae7aab3f0005a652fd66b6f57))
* **secrets:** resolve bw serve items from the unfiltered list, not the search index ([#1822](https://github.com/mlorentedev/dotfiles/issues/1822)) ([c719016](https://github.com/mlorentedev/dotfiles/commit/c719016ae1691c5df87820ae9270ac45d253f1fc))
* **spec-gate:** exclude patch-tool output from production LOC ([#1944](https://github.com/mlorentedev/dotfiles/issues/1944)) ([b664be6](https://github.com/mlorentedev/dotfiles/commit/b664be60043cd25569463c5f1a25d0052e314d02))
* **spec:** avoid agy sandbox elevation on Windows ([#1840](https://github.com/mlorentedev/dotfiles/issues/1840)) ([db6e30f](https://github.com/mlorentedev/dotfiles/commit/db6e30f7aeac77f7595c7ae5591b6ed4c4c15ff6))
* **spec:** reject malformed review artifacts ([#1828](https://github.com/mlorentedev/dotfiles/issues/1828)) ([cba58e6](https://github.com/mlorentedev/dotfiles/commit/cba58e6f6c8aa3fd6cb97240e5faeee3b2fe0d4a))
* **specs:** AI-044's parity evidence runs the test that holds the assertion ([#1867](https://github.com/mlorentedev/dotfiles/issues/1867)) ([8a12783](https://github.com/mlorentedev/dotfiles/commit/8a127832da7a034b963353973ab0fcad2a9a2a49))
* **tools:** probe installed versions with the same rule as tools version ([#1900](https://github.com/mlorentedev/dotfiles/issues/1900)) ([2356fd4](https://github.com/mlorentedev/dotfiles/commit/2356fd42563af2f4925f6ffa754e9786ad98e821))

## [0.61.0](https://github.com/mlorentedev/dotfiles/compare/v0.60.0...v0.61.0) (2026-09-29)


### Features

* **pi:** let pi-nan-provider own the NaN model ids ([#1789](https://github.com/mlorentedev/dotfiles/issues/1789)) ([c114094](https://github.com/mlorentedev/dotfiles/commit/c114094003855544fe1d399d4dd996064a2ae52b))
* **secrets:** curate the items the registry does not declare from a reviewed plan ([#1790](https://github.com/mlorentedev/dotfiles/issues/1790)) ([5fdf472](https://github.com/mlorentedev/dotfiles/commit/5fdf47219476360f9b1e6ef10f43de50311f2871))


### Bug Fixes

* **bitacora:** resolve the project id in every rollout mode, not only --backfill-only ([#1788](https://github.com/mlorentedev/dotfiles/issues/1788)) ([98f145e](https://github.com/mlorentedev/dotfiles/commit/98f145ea1fcbd4c0c1f4d6dc888203c2926d114b)), closes [#1786](https://github.com/mlorentedev/dotfiles/issues/1786)
* **doctor:** name an unrouted default model as unrouted, not missing ([#1816](https://github.com/mlorentedev/dotfiles/issues/1816)) ([d686558](https://github.com/mlorentedev/dotfiles/commit/d6865589e75208ca7518e869f1b5d396ebeb7a62))
* **doctor:** tell a committed orphan age blob from an untracked copy ([#1807](https://github.com/mlorentedev/dotfiles/issues/1807)) ([7e97e96](https://github.com/mlorentedev/dotfiles/commit/7e97e9663ded83d5c5ecc3c227ebc34c0cbcd90f))
* **harness:** make mirror checkout explicit ([#1806](https://github.com/mlorentedev/dotfiles/issues/1806)) ([79c89b4](https://github.com/mlorentedev/dotfiles/commit/79c89b438a60e699ad9f34ac20ed39a3ec620fec))
* **installer:** support checkout-free release recovery ([#1805](https://github.com/mlorentedev/dotfiles/issues/1805)) ([66a90b7](https://github.com/mlorentedev/dotfiles/commit/66a90b758733e109961a803f0a94cfcb30efe393))
* **lint:** enable the gofmt formatter and format the four files it finds ([#1797](https://github.com/mlorentedev/dotfiles/issues/1797)) ([b15ad97](https://github.com/mlorentedev/dotfiles/commit/b15ad970f98c1ec6de1a2e51defc944cbbb0280a)), closes [#1154](https://github.com/mlorentedev/dotfiles/issues/1154)
* **secrets:** curate refuses a write to a registry-declared merge keeper ([#1812](https://github.com/mlorentedev/dotfiles/issues/1812)) ([51e3638](https://github.com/mlorentedev/dotfiles/commit/51e3638ece29ca51ee5b80f669bf5f8f25391f13))
* **secrets:** force the bw serve sync so the daemon never serves a stale cache ([#1821](https://github.com/mlorentedev/dotfiles/issues/1821)) ([9d611cb](https://github.com/mlorentedev/dotfiles/commit/9d611cb9f92bae0524d13c54c70693ff3bbeb600))
* **sync:** stop dotfiles-sync from copying sensitive/ and stop test.sh running it ([#1796](https://github.com/mlorentedev/dotfiles/issues/1796)) ([c2fed9b](https://github.com/mlorentedev/dotfiles/commit/c2fed9bb0c9855b232fc3b0aedca7f5cb86f9f13)), closes [#1795](https://github.com/mlorentedev/dotfiles/issues/1795)

## [0.60.0](https://github.com/mlorentedev/dotfiles/compare/v0.59.0...v0.60.0) (2026-09-27)


### Features

* **claude:** set language to spanish so voice dictation is Spanish ([#1771](https://github.com/mlorentedev/dotfiles/issues/1771)) ([88a6591](https://github.com/mlorentedev/dotfiles/commit/88a6591cf94319df7b2d0e378abeacdd3a3c712b))
* **doctor:** warn before a bound NaN model runs out of quota ([#1779](https://github.com/mlorentedev/dotfiles/issues/1779)) ([f917eff](https://github.com/mlorentedev/dotfiles/commit/f917effb32c86b5bedbb6a695b01f9e1fe89f1ea))
* **forge:** apply declared branch protection, and require spec-gate on dotfiles ([#1746](https://github.com/mlorentedev/dotfiles/issues/1746)) ([60de574](https://github.com/mlorentedev/dotfiles/commit/60de5746ef7fd7247ff0cc272cdcf4ee0676993c))
* **harness:** add a knowledge gate that checks a PR names its lesson, ADR and runbook ([#1732](https://github.com/mlorentedev/dotfiles/issues/1732)) ([6ceebd4](https://github.com/mlorentedev/dotfiles/commit/6ceebd46e4869bf5c98a32d358eddba7a4ed6496))
* **harness:** the handoff skill passes the calling agent's own name, and MEMORY-009 archives ([#1729](https://github.com/mlorentedev/dotfiles/issues/1729)) ([b2364ab](https://github.com/mlorentedev/dotfiles/commit/b2364ab3ff2ac4b445575fc4983688189e2624bc))
* **pi:** dotf pi packages converges pi on its manifest both ways, and both setups call it ([#1754](https://github.com/mlorentedev/dotfiles/issues/1754)) ([e4b39f5](https://github.com/mlorentedev/dotfiles/commit/e4b39f553b6b3357495c2ce2873f61f8abbcd4f4))
* **pi:** install pi-nan-provider with its media MCP bridge off ([#1778](https://github.com/mlorentedev/dotfiles/issues/1778)) ([b7ebd3e](https://github.com/mlorentedev/dotfiles/commit/b7ebd3e2b2422386891a8f0fd74205c7f3587208))
* **pi:** the manifest retires pi-memory data to an archive, and doctor checks declared requirements ([#1755](https://github.com/mlorentedev/dotfiles/issues/1755)) ([4dad5ea](https://github.com/mlorentedev/dotfiles/commit/4dad5eadd8857c084499e2474a67b4ac87024ea7))
* **skills:** add a test authoring gate and a deletion evidence bar, adapted from OpenClaw test-audit ([#1777](https://github.com/mlorentedev/dotfiles/issues/1777)) ([bb5d12d](https://github.com/mlorentedev/dotfiles/commit/bb5d12d30cb6b7926c58b4bd11f417efdd4d1bf0))
* **spec:** refuse an archive whose promotion candidates are not answered ([#1761](https://github.com/mlorentedev/dotfiles/issues/1761)) ([030ade1](https://github.com/mlorentedev/dotfiles/commit/030ade1e1ca5539b1adf7d38eae20ea3524fd957))


### Bug Fixes

* **ci:** the release PR body closes nothing ([#1734](https://github.com/mlorentedev/dotfiles/issues/1734)) ([57f81c7](https://github.com/mlorentedev/dotfiles/commit/57f81c75260e4d9e33c5642e94579bf9524545f5))
* **lessons:** stop the index check reporting indexed lessons as missing at random ([#1736](https://github.com/mlorentedev/dotfiles/issues/1736)) ([f9d5b25](https://github.com/mlorentedev/dotfiles/commit/f9d5b25e9aea30612347c1f581c4ccc93a04067b))
* **mem:** move text before the first thread only when it reads as a handoff ([#1726](https://github.com/mlorentedev/dotfiles/issues/1726)) ([c17c4ce](https://github.com/mlorentedev/dotfiles/commit/c17c4ceeebb73fca242627048df9ab6eba4134db))
* **opencode:** move routed traffic off qwen3.8-flash before its NaN quota runs out ([#1772](https://github.com/mlorentedev/dotfiles/issues/1772)) ([75eb855](https://github.com/mlorentedev/dotfiles/commit/75eb855d49001872f97af666b4d931ee26f3487a))
* **pr-agent:** only the bot's own review sets the push gate's baseline, and a rebase is reviewed in full ([#1773](https://github.com/mlorentedev/dotfiles/issues/1773)) ([94c9e6b](https://github.com/mlorentedev/dotfiles/commit/94c9e6bc80ef4f3d55374ca1cf156cac889d7f8a))
* **secrets:** retire the 31 legacy age blobs; doctor fails and prunes unclaimed ones ([#1776](https://github.com/mlorentedev/dotfiles/issues/1776)) ([6c963fa](https://github.com/mlorentedev/dotfiles/commit/6c963fa7c64771fcab04f3d02553bf4ae7224120))
* **spec:** accept hyphen-joined AREAs in feature ids ([#1479](https://github.com/mlorentedev/dotfiles/issues/1479)) ([#1769](https://github.com/mlorentedev/dotfiles/issues/1769)) ([37d7e64](https://github.com/mlorentedev/dotfiles/commit/37d7e64c46e72449e50e139b05d445e604ee4527))

## [0.59.0](https://github.com/mlorentedev/dotfiles/compare/v0.58.0...v0.59.0) (2026-09-25)


### Features

* automate Windows SSH key recovery ([#1659](https://github.com/mlorentedev/dotfiles/issues/1659)) ([8636c58](https://github.com/mlorentedev/dotfiles/commit/8636c5847109c8599b2e2a5b00f10bdc9eb19111))
* **harness:** slim three persona rosters and fold eight skills into the ones that remain ([#1694](https://github.com/mlorentedev/dotfiles/issues/1694)) ([3c2e393](https://github.com/mlorentedev/dotfiles/commit/3c2e393e9fdfbc3b3aa89f2e0b40f90cb62aae4f))
* **mem:** handoff-write --agent keeps another agent's thread and forks the write ([#1711](https://github.com/mlorentedev/dotfiles/issues/1711)) ([6436c71](https://github.com/mlorentedev/dotfiles/commit/6436c7167dccef33704c01de2d9f26cc9dd81b7c))
* **mem:** read a handoff thread into its canonical fields and warn on the ones it lacks ([#1716](https://github.com/mlorentedev/dotfiles/issues/1716)) ([ffe5c44](https://github.com/mlorentedev/dotfiles/commit/ffe5c44b515eee92ba8e245c1ca10f73537443ea))


### Bug Fixes

* **deploy:** finish AI-042 trusted folder rendering ([#1723](https://github.com/mlorentedev/dotfiles/issues/1723)) ([cb0313a](https://github.com/mlorentedev/dotfiles/commit/cb0313a0ee9c23c8da967dca71af0ff50a6e7789))
* **harness:** read skill prerequisites from the deployed records instead of the map compiled into the binary ([#1714](https://github.com/mlorentedev/dotfiles/issues/1714)) ([b543552](https://github.com/mlorentedev/dotfiles/commit/b543552b189736cc44c643add14e2260cf5f0256))
* **harness:** the gate reserves the unparsed journal's name as it reserves the unscoped one ([#1707](https://github.com/mlorentedev/dotfiles/issues/1707)) ([1c33456](https://github.com/mlorentedev/dotfiles/commit/1c334560147cda30240c0d9b4e7c98f598e09e8c))
* **harness:** the trigger guards refuse a file they could not read or that names no pattern ([#1706](https://github.com/mlorentedev/dotfiles/issues/1706)) ([355a784](https://github.com/mlorentedev/dotfiles/commit/355a784e0e1c0211c3c81fa18a8f3801c013f96b))
* **mem:** handoff-write moves an un-threaded handoff block into a legacy thread ([#1703](https://github.com/mlorentedev/dotfiles/issues/1703)) ([221e0da](https://github.com/mlorentedev/dotfiles/commit/221e0daf8bea47d1412d852214a07871992f01b9))
* **mem:** handoff-write refuses a thread key read from another repository ([#1702](https://github.com/mlorentedev/dotfiles/issues/1702)) ([8ce4bcd](https://github.com/mlorentedev/dotfiles/commit/8ce4bcd4f01ea3bac10c91e521e02cee6f1a45ad))
* **mem:** session-end writes its record only where no journal exists ([#1701](https://github.com/mlorentedev/dotfiles/issues/1701)) ([9603e9a](https://github.com/mlorentedev/dotfiles/commit/9603e9acb1a3e9796425579f79d09a32f41c125d))
* **spec:** bound every reviewer runner with the review deadline and tell the reviewer its time budget ([#1721](https://github.com/mlorentedev/dotfiles/issues/1721)) ([58fb872](https://github.com/mlorentedev/dotfiles/commit/58fb8726985758778ffb65ac873f7f58c1b3d9b9))
* **spec:** the contract digest folds the lifecycle status the archive writes ([#1709](https://github.com/mlorentedev/dotfiles/issues/1709)) ([29947d1](https://github.com/mlorentedev/dotfiles/commit/29947d1a9976656c60e28a7e9b665d23981d2be2))

## [0.58.0](https://github.com/mlorentedev/dotfiles/compare/v0.57.0...v0.58.0) (2026-09-25)


### Features

* **harness:** read agy's own hook payload and register its gate where agy reads it ([#1688](https://github.com/mlorentedev/dotfiles/issues/1688)) ([cd619b5](https://github.com/mlorentedev/dotfiles/commit/cd619b59121287a1665096eb4d17ac2c259c8dd9))
* **secrets:** give the personal plane its folder, Dotfiles/personal, one folder per plane ([#1673](https://github.com/mlorentedev/dotfiles/issues/1673)) ([08b8e19](https://github.com/mlorentedev/dotfiles/commit/08b8e19268a462972b46bd7e39f1cfd4167b130d)), closes [#586](https://github.com/mlorentedev/dotfiles/issues/586)


### Bug Fixes

* **harness:** point every trigger at a pattern that exists and refuse a dangling one ([#1687](https://github.com/mlorentedev/dotfiles/issues/1687)) ([1f833f4](https://github.com/mlorentedev/dotfiles/commit/1f833f4afb8cbb991094bbef108b624e5bedf1d0))
* **harness:** slim the enforced doctrine and fold every capped surface to ASCII ([#1685](https://github.com/mlorentedev/dotfiles/issues/1685)) ([ebc2e20](https://github.com/mlorentedev/dotfiles/commit/ebc2e20c3eacb53730f9f50eb1986a38bcd8c0fe))
* **harness:** stop payloads with no session id sharing one gate ledger ([#1683](https://github.com/mlorentedev/dotfiles/issues/1683)) ([3a8e52a](https://github.com/mlorentedev/dotfiles/commit/3a8e52aa936880b3fc745d1fc38aa41615fda848))
* **secrets:** recognise every harness's session marker, refuse before decrypting, and fail closed on shell snippets ([#1677](https://github.com/mlorentedev/dotfiles/issues/1677)) ([bfcc072](https://github.com/mlorentedev/dotfiles/commit/bfcc072239941f3dd54c9051b2761fa549afeff4))
* **secrets:** render the locked-vault backup remedy from the invocation that failed ([#1662](https://github.com/mlorentedev/dotfiles/issues/1662)) ([be68b63](https://github.com/mlorentedev/dotfiles/commit/be68b63e7f47abb7071896b5ba4f857fa1602f69)), closes [#1647](https://github.com/mlorentedev/dotfiles/issues/1647)

## [0.57.0](https://github.com/mlorentedev/dotfiles/compare/v0.56.0...v0.57.0) (2026-09-24)


### Features

* **forge:** declare branch protection in git and check every repository for drift ([#1637](https://github.com/mlorentedev/dotfiles/issues/1637)) ([da16c65](https://github.com/mlorentedev/dotfiles/commit/da16c65608ce4a56f79b9d4ca79760651218483f)), closes [#1451](https://github.com/mlorentedev/dotfiles/issues/1451)
* **pi:** connect pi to hive and mcp servers via pi-mcp-client ([#1657](https://github.com/mlorentedev/dotfiles/issues/1657)) ([32c9fe2](https://github.com/mlorentedev/dotfiles/commit/32c9fe2c9a979dac6c154c8a49ffd93709c34261))
* **secrets:** retire a dead field by declaration, and take bw.from on a multi-var secret over one field ([#1652](https://github.com/mlorentedev/dotfiles/issues/1652)) ([89f2646](https://github.com/mlorentedev/dotfiles/commit/89f26468d6126c7e1d139140f5f387e15d95124b))
* **secrets:** verdicts for every retire in the reconcile plan, and retiring items by declaration ([#1634](https://github.com/mlorentedev/dotfiles/issues/1634)) ([37e3ddc](https://github.com/mlorentedev/dotfiles/commit/37e3ddccfe33f2168612991318effaa0dee56038)), closes [#1624](https://github.com/mlorentedev/dotfiles/issues/1624)
* **spec:** audit active specs against the state of the issue they track ([#1630](https://github.com/mlorentedev/dotfiles/issues/1630)) ([2c6af81](https://github.com/mlorentedev/dotfiles/commit/2c6af81c8a4b581ace62e45c0ca824b15017f913)), closes [#1087](https://github.com/mlorentedev/dotfiles/issues/1087)
* **spec:** decide review freshness by content, and record every archive bypass ([#1633](https://github.com/mlorentedev/dotfiles/issues/1633)) ([fced814](https://github.com/mlorentedev/dotfiles/commit/fced814ce66650e6e9d428096ebd3e4242cac401)), closes [#1566](https://github.com/mlorentedev/dotfiles/issues/1566)


### Bug Fixes

* **ci:** skip PR-Agent when every file is ignored, instead of blaming NaN ([#1619](https://github.com/mlorentedev/dotfiles/issues/1619)) ([19de983](https://github.com/mlorentedev/dotfiles/commit/19de983340e54393d088cc52bf83a5183daf20df)), closes [#1618](https://github.com/mlorentedev/dotfiles/issues/1618)
* **doctor:** gate PAT resolution on the readability probe, not GET /status ([#1612](https://github.com/mlorentedev/dotfiles/issues/1612)) ([0642e31](https://github.com/mlorentedev/dotfiles/commit/0642e315770765d52632585dbc1f571e2fa313d2)), closes [#1611](https://github.com/mlorentedev/dotfiles/issues/1611)
* **secrets:** converge copy+retire in one apply, and check what drift used to skip ([#1621](https://github.com/mlorentedev/dotfiles/issues/1621)) ([8a68a49](https://github.com/mlorentedev/dotfiles/commit/8a68a49c01b38bac6694eda4eb87aeb15b63e67a)), closes [#1596](https://github.com/mlorentedev/dotfiles/issues/1596)
* **secrets:** refuse env dumps behind a path or a redirect, and detect Claude Code's real session marker ([#1655](https://github.com/mlorentedev/dotfiles/issues/1655)) ([02f9165](https://github.com/mlorentedev/dotfiles/commit/02f9165c97edcb6adb1ef8f852c64945bd3d8b6d)), closes [#1626](https://github.com/mlorentedev/dotfiles/issues/1626)
* **secrets:** take a file secret as hidden multi-line input instead of advising a file on disk ([#1639](https://github.com/mlorentedev/dotfiles/issues/1639)) ([79b9a13](https://github.com/mlorentedev/dotfiles/commit/79b9a1301a24877c3afcedaab9201b0f524d6ee7)), closes [#1638](https://github.com/mlorentedev/dotfiles/issues/1638)
* **spec:** declare review state as one exported list, including review-request.json ([#1631](https://github.com/mlorentedev/dotfiles/issues/1631)) ([c679d41](https://github.com/mlorentedev/dotfiles/commit/c679d41ca3b1dd1042678d162a6eeb89d5ef556a)), closes [#998](https://github.com/mlorentedev/dotfiles/issues/998)
* **tools:** stop installing an unrelated npm package as the Obsidian CLI ([#1616](https://github.com/mlorentedev/dotfiles/issues/1616)) ([52e99a2](https://github.com/mlorentedev/dotfiles/commit/52e99a2d595d138c1b4f1197babe5772d7ceeebc)), closes [#1615](https://github.com/mlorentedev/dotfiles/issues/1615)

## [0.56.0](https://github.com/mlorentedev/dotfiles/compare/v0.55.0...v0.56.0) (2026-09-23)


### Features

* **secrets:** drift fixes, and converge the Bitwarden layout with dotf secrets reconcile ([#1600](https://github.com/mlorentedev/dotfiles/issues/1600)) ([32eb6fc](https://github.com/mlorentedev/dotfiles/commit/32eb6fc5a8060215b5cdcc252a06e902b0ecdaf8)), closes [#1596](https://github.com/mlorentedev/dotfiles/issues/1596)
* **secrets:** report where the Bitwarden store disagrees with the registry ([#1597](https://github.com/mlorentedev/dotfiles/issues/1597)) ([799ca66](https://github.com/mlorentedev/dotfiles/commit/799ca6663a8647ff3158ff1504c60c9971c4c506))


### Bug Fixes

* **doctor:** fail on an unreadable model declaration, and stop counting nothing as a match ([#1602](https://github.com/mlorentedev/dotfiles/issues/1602)) ([33c3aa7](https://github.com/mlorentedev/dotfiles/commit/33c3aa7bf94b07a173632059371d2e1331b4c007)), closes [#1594](https://github.com/mlorentedev/dotfiles/issues/1594)
* **harness:** new-ticket verified the board with a listing that truncates silently ([#1587](https://github.com/mlorentedev/dotfiles/issues/1587)) ([3b6a18c](https://github.com/mlorentedev/dotfiles/commit/3b6a18cd390b389aec203855c4a34dd96207fc76))
* **pi:** align model limits with the provider catalog, and report future drift ([#1595](https://github.com/mlorentedev/dotfiles/issues/1595)) ([c52e637](https://github.com/mlorentedev/dotfiles/commit/c52e637cd2e014d5c897853f74841dbdc1061f66))

## [0.55.0](https://github.com/mlorentedev/dotfiles/compare/v0.54.0...v0.55.0) (2026-09-07)


### Features

* **agent:** dispatch the persona a task implies, on the tier it declares ([#1546](https://github.com/mlorentedev/dotfiles/issues/1546)) ([23ffc3c](https://github.com/mlorentedev/dotfiles/commit/23ffc3c8c25f75f60182488cd6856b035b441873)), closes [#1537](https://github.com/mlorentedev/dotfiles/issues/1537)
* **harness:** curator becomes the fifth enforceable persona, gating two of eight ([#1527](https://github.com/mlorentedev/dotfiles/issues/1527)) ([5101938](https://github.com/mlorentedev/dotfiles/commit/5101938aa0013cb2105acedb706f2d02754580cb))
* **harness:** planner becomes the sixth enforceable persona, gating one of six ([#1528](https://github.com/mlorentedev/dotfiles/issues/1528)) ([1b57cc0](https://github.com/mlorentedev/dotfiles/commit/1b57cc0c4b698be288590ba56743dbc89b37e2b8))
* **spec:** give the adversarial review a base to diff against, and let the verdict use its own reality axis ([#1535](https://github.com/mlorentedev/dotfiles/issues/1535)) ([5a68096](https://github.com/mlorentedev/dotfiles/commit/5a68096fb19d9d6fb64b2abca033d8f37ce579ab))


### Bug Fixes

* **ci:** bind the PR-Agent guard to its author and to this run ([#1567](https://github.com/mlorentedev/dotfiles/issues/1567)) ([b4660a7](https://github.com/mlorentedev/dotfiles/commit/b4660a70193ffb3f688b7dc989b08adec6807150))
* **cli:** count a reviewer's in-place edit as new output in the triage queue ([#1557](https://github.com/mlorentedev/dotfiles/issues/1557)) ([48537c4](https://github.com/mlorentedev/dotfiles/commit/48537c4f223291648b0eed739671b09dd5bd179e)), closes [#1422](https://github.com/mlorentedev/dotfiles/issues/1422)
* **cli:** make Gate f's claims falsifiable, and stop calling a constant a signal ([#1526](https://github.com/mlorentedev/dotfiles/issues/1526)) ([195cc18](https://github.com/mlorentedev/dotfiles/commit/195cc1887b9ac1c9f121589cc609e560f730fedc))
* **cli:** run Gate f last, immediately before the removal ([#1531](https://github.com/mlorentedev/dotfiles/issues/1531)) ([36c27fb](https://github.com/mlorentedev/dotfiles/commit/36c27fb15a6fdebbd56bac4f5daa2d7ad378b1f8))
* **harness:** qualify the Major severity definition by reality, and make f7 assert ([#1549](https://github.com/mlorentedev/dotfiles/issues/1549)) ([53e9c84](https://github.com/mlorentedev/dotfiles/commit/53e9c846ff2f6383a93bf3288a1f916a3fe7b106))
* **harness:** shipper's block precondition named a closed issue and the wrong risk ([#1524](https://github.com/mlorentedev/dotfiles/issues/1524)) ([43d4b0c](https://github.com/mlorentedev/dotfiles/commit/43d4b0ca85f274ce4ee1ff8c5af10c3330aa86f0)), closes [#1497](https://github.com/mlorentedev/dotfiles/issues/1497)
* **harness:** stop the review skill asking for the edits the archive gate refuses ([#1543](https://github.com/mlorentedev/dotfiles/issues/1543)) ([39754ec](https://github.com/mlorentedev/dotfiles/commit/39754ec80efecd772570d9c2c2262dd738a3b7d2))
* **harness:** the gate's allow reason was vacuously true, so an inert gate read like a passing one ([#1534](https://github.com/mlorentedev/dotfiles/issues/1534)) ([1fe3531](https://github.com/mlorentedev/dotfiles/commit/1fe3531798120056a75fb9b2d66741ec1635e46f)), closes [#1510](https://github.com/mlorentedev/dotfiles/issues/1510)
* **lessons:** main is red — renumber two colliding lessons to 279 and 280 ([#1568](https://github.com/mlorentedev/dotfiles/issues/1568)) ([d38f5dc](https://github.com/mlorentedev/dotfiles/commit/d38f5dccd2171737a252c32617a01e09dbb86d54)), closes [#1542](https://github.com/mlorentedev/dotfiles/issues/1542)
* **memlink:** map '.' to '-' in the Claude project key ([#1560](https://github.com/mlorentedev/dotfiles/issues/1560)) ([12a015b](https://github.com/mlorentedev/dotfiles/commit/12a015b15555ac827bcd56ee19aff8a7e392e64a))
* **pi:** drop an input modality pi rejects, which was taking out every model ([#1539](https://github.com/mlorentedev/dotfiles/issues/1539)) ([688db44](https://github.com/mlorentedev/dotfiles/commit/688db4482d17648196834fa2ce2e28926f80edc5))
* **tests:** stop two bats suites reading the developer's machine ([#1556](https://github.com/mlorentedev/dotfiles/issues/1556)) ([18095f2](https://github.com/mlorentedev/dotfiles/commit/18095f203d21b68e3ab753cbf2a5b55307734cba))

## [0.54.0](https://github.com/mlorentedev/dotfiles/compare/v0.53.0...v0.54.0) (2026-09-05)


### Features

* **cli:** dotf vault maintain, composing crystallize and health in-process ([#1489](https://github.com/mlorentedev/dotfiles/issues/1489)) ([52035f1](https://github.com/mlorentedev/dotfiles/commit/52035f1a174e82f3176bc458aa128ebc77909f9e)), closes [#490](https://github.com/mlorentedev/dotfiles/issues/490)
* **cli:** dotf worktree lifecycle and fail-closed garbage collection ([#1500](https://github.com/mlorentedev/dotfiles/issues/1500)) ([#1515](https://github.com/mlorentedev/dotfiles/issues/1515)) ([a76ea18](https://github.com/mlorentedev/dotfiles/commit/a76ea184ea20f8f54903df15c7d558074f22e241))
* **doctor:** print a Next steps block for FAILs that carry a remedy command ([#1443](https://github.com/mlorentedev/dotfiles/issues/1443)) ([1966d6d](https://github.com/mlorentedev/dotfiles/commit/1966d6da24b90e7d4517abfbe3834f83a01d9bf7))
* **doctor:** the binary carries the commit it was built from, so a stale dotf stops reporting health it never established ([#1416](https://github.com/mlorentedev/dotfiles/issues/1416)) ([3a11d97](https://github.com/mlorentedev/dotfiles/commit/3a11d977d6c3de7cbb3f57a8063ec8c6a65d7695)), closes [#1158](https://github.com/mlorentedev/dotfiles/issues/1158)
* **harness:** architect becomes the fourth enforceable persona, and a typo in any skill id now fails ([#1509](https://github.com/mlorentedev/dotfiles/issues/1509)) ([4e2e2c4](https://github.com/mlorentedev/dotfiles/commit/4e2e2c4d2551163a9ff633bfaac91b2c94331810))
* **harness:** builder becomes the second enforceable persona, gating two of nine ([#1494](https://github.com/mlorentedev/dotfiles/issues/1494)) ([ca81157](https://github.com/mlorentedev/dotfiles/commit/ca8115734e08677f72f19f649ff2259587052d86)), closes [#1420](https://github.com/mlorentedev/dotfiles/issues/1420)
* **harness:** fire the persona suggester on every prompt, failing open by construction ([#1455](https://github.com/mlorentedev/dotfiles/issues/1455)) ([fb0b7e9](https://github.com/mlorentedev/dotfiles/commit/fb0b7e9219270a8fc53f51d1757ce1a6ad2baccc))
* **harness:** grant personas the capability to invoke the skills their gate demands ([#1428](https://github.com/mlorentedev/dotfiles/issues/1428)) ([16d8f96](https://github.com/mlorentedev/dotfiles/commit/16d8f9634732b01a1bbb1b9a8357f3532627f1e0))
* **harness:** shipper becomes the third enforceable persona, and adopts an orphaned skill ([#1504](https://github.com/mlorentedev/dotfiles/issues/1504)) ([5cc14ba](https://github.com/mlorentedev/dotfiles/commit/5cc14bad4c1debec91b22a74f2f99e2999d3fa29)), closes [#1497](https://github.com/mlorentedev/dotfiles/issues/1497)
* **harness:** the gate writes down every decision it takes ([#1435](https://github.com/mlorentedev/dotfiles/issues/1435)) ([8fecec6](https://github.com/mlorentedev/dotfiles/commit/8fecec6736dc34b0447e418e8808af53d9f6f0d5))
* **hooks:** dotf hooks install, retiring the install-git-hooks twin pair and its test twins ([#1464](https://github.com/mlorentedev/dotfiles/issues/1464)) ([c375102](https://github.com/mlorentedev/dotfiles/commit/c375102a124de6f42f8fd21909e912a1ec00eb53)), closes [#1460](https://github.com/mlorentedev/dotfiles/issues/1460)
* **secrets:** guard secrets run against environment introspection commands ([#1459](https://github.com/mlorentedev/dotfiles/issues/1459)) ([a720b9d](https://github.com/mlorentedev/dotfiles/commit/a720b9d2b40569b73c96aadc5ceadbd8a41d8f7f))


### Bug Fixes

* **cli:** make Gate f fail closed where it has no implementation ([#1518](https://github.com/mlorentedev/dotfiles/issues/1518)) ([6f9b0d9](https://github.com/mlorentedev/dotfiles/commit/6f9b0d95729efbe853e6acabefa2f67863cb94f8))
* **docs:** give every lesson its own number, and a guard so the next one cannot land ([#1514](https://github.com/mlorentedev/dotfiles/issues/1514)) ([#1519](https://github.com/mlorentedev/dotfiles/issues/1519)) ([4d0ffa9](https://github.com/mlorentedev/dotfiles/commit/4d0ffa99f7612797add2eb9d945634ecb376cf8d))
* **doctor:** pi/opencode/copilot/yarn version-pin checks agree with their installer's floor policy ([#1441](https://github.com/mlorentedev/dotfiles/issues/1441)) ([7a81ffe](https://github.com/mlorentedev/dotfiles/commit/7a81ffea0592df58e6bf8ae211af4d0937c4dec4))
* **harness:** land what [#1455](https://github.com/mlorentedev/dotfiles/issues/1455) merged without, and archive the spec it closed ([#1462](https://github.com/mlorentedev/dotfiles/issues/1462)) ([0105662](https://github.com/mlorentedev/dotfiles/commit/0105662daf0c04c77f43adc9b02f588604b148c6))
* **harness:** resolve a named dispatch through the dispatch that created it ([#1471](https://github.com/mlorentedev/dotfiles/issues/1471)) ([d99a31e](https://github.com/mlorentedev/dotfiles/commit/d99a31e230e605301da3baf4997c63bcdaf533b7)), closes [#1434](https://github.com/mlorentedev/dotfiles/issues/1434)
* **harness:** scan ids over REST and yield on ownership, not on issue number ([#1463](https://github.com/mlorentedev/dotfiles/issues/1463)) ([fe4a4ab](https://github.com/mlorentedev/dotfiles/commit/fe4a4ab5f1649cd0bf6f47f4bca1983a2fd33f62))
* **harness:** stamp the new-ticket record with its source's actual hash ([#1503](https://github.com/mlorentedev/dotfiles/issues/1503)) ([ecd20ef](https://github.com/mlorentedev/dotfiles/commit/ecd20efed698d4ccb82ef19c78f00c67ae4f836c))
* **harness:** the crystallize skill told agents to run a script deleted in [#1276](https://github.com/mlorentedev/dotfiles/issues/1276) ([#1490](https://github.com/mlorentedev/dotfiles/issues/1490)) ([362783b](https://github.com/mlorentedev/dotfiles/commit/362783bb201af92f275b40f37c636f1827d8ea3c))
* **harness:** the doctrine cap was asserted in characters while the payload was 47 bytes over it ([#1513](https://github.com/mlorentedev/dotfiles/issues/1513)) ([f028d12](https://github.com/mlorentedev/dotfiles/commit/f028d12c9458943d4c8c7976dea0855742025dd2))
* **memlink:** quote mklink's arguments for cmd.exe's own tokenizer, not Go's argv escaping ([#1444](https://github.com/mlorentedev/dotfiles/issues/1444)) ([792912f](https://github.com/mlorentedev/dotfiles/commit/792912feb76a63ee43aca2c7903e45925e3cca7d))
* **prtriage:** give the fetch path a seam, then move it off GraphQL ([#1457](https://github.com/mlorentedev/dotfiles/issues/1457)) ([afc9db6](https://github.com/mlorentedev/dotfiles/commit/afc9db6021763589f695c9bf9e1ecc898d4d1177)), closes [#1454](https://github.com/mlorentedev/dotfiles/issues/1454)
* **secrets:** give an interactive child a pty, so secrets run stops breaking TUIs ([#1508](https://github.com/mlorentedev/dotfiles/issues/1508)) ([46dc9d8](https://github.com/mlorentedev/dotfiles/commit/46dc9d8b199b257f0ff1548937aa656d44e67eed))
* **setup:** log what a pi install cost and what it printed, in both twins ([#1488](https://github.com/mlorentedev/dotfiles/issues/1488)) ([173f788](https://github.com/mlorentedev/dotfiles/commit/173f78891a08cc7939bfc8378e03596838064953)), closes [#1486](https://github.com/mlorentedev/dotfiles/issues/1486)

## [0.53.0](https://github.com/mlorentedev/dotfiles/compare/v0.52.0...v0.53.0) (2026-09-01)


### Features

* **cli:** HARNESS-105 refactor shellQuote and implement JSON Error Latches ([#1404](https://github.com/mlorentedev/dotfiles/issues/1404)) ([d4ea0f5](https://github.com/mlorentedev/dotfiles/commit/d4ea0f571e3e81daf25847b121e9d2e703e18f42))
* **harness:** bind hooks through dotf, removing both setup scripts' hooks writers ([#1407](https://github.com/mlorentedev/dotfiles/issues/1407)) ([fdbe17c](https://github.com/mlorentedev/dotfiles/commit/fdbe17c8cdeac29837a56cddb3bc4bb60219a1a6)), closes [#561](https://github.com/mlorentedev/dotfiles/issues/561)
* **harness:** migrate reviewer to declared severity, and teach the roster guard the new form ([#1412](https://github.com/mlorentedev/dotfiles/issues/1412)) ([cd7416f](https://github.com/mlorentedev/dotfiles/commit/cd7416ff64c99aec15f137b527c55b20455123d9))
* **harness:** the gate resolves its persona from the payload, and doctor reports what it cannot enforce ([#1410](https://github.com/mlorentedev/dotfiles/issues/1410)) ([2dbaa06](https://github.com/mlorentedev/dotfiles/commit/2dbaa064d23ba79bc859816817bb6a8b9c7d272d))
* **setup:** make the no-attribution order mechanical and stop holding peer messages ([#1411](https://github.com/mlorentedev/dotfiles/issues/1411)) ([a7364e1](https://github.com/mlorentedev/dotfiles/commit/a7364e1a20e7691de78359802c51ba878ffdec7a))


### Bug Fixes

* **deploy:** agy owns the settings it writes, so the deploy stops replacing them ([#1414](https://github.com/mlorentedev/dotfiles/issues/1414)) ([6e0d180](https://github.com/mlorentedev/dotfiles/commit/6e0d180ae57c24148fbe3e74af7c7f75d86c40f1))
* **setup:** restore setup-linux.sh executable bit and guard the invocation ([#1413](https://github.com/mlorentedev/dotfiles/issues/1413)) ([fd02554](https://github.com/mlorentedev/dotfiles/commit/fd02554e3528781ecd563eaae5f2549c0dd1254d))

## [0.52.0](https://github.com/mlorentedev/dotfiles/compare/v0.51.0...v0.52.0) (2026-08-30)


### Features

* **cli:** dotf vault health, byte-identical to the shell oracle (CLI-021 increment 2) ([#1317](https://github.com/mlorentedev/dotfiles/issues/1317)) ([7a06364](https://github.com/mlorentedev/dotfiles/commit/7a06364363d36d897500f3a71a6bd6e99dfbb21d))
* **deploy:** a 0600 file gets an owner-only ACL on Windows, and an in-sync file still gets its declared mode ([#1380](https://github.com/mlorentedev/dotfiles/issues/1380)) ([3f554ad](https://github.com/mlorentedev/dotfiles/commit/3f554ad73aa8253a09b5dc926abb26599de408c3))
* **deploy:** ai/deploy.json learns merge and requires, and Copilot's JSON moves onto it ([#1365](https://github.com/mlorentedev/dotfiles/issues/1365)) ([3261d04](https://github.com/mlorentedev/dotfiles/commit/3261d0431f82edd09fbf6c1d078b929b21558088))
* **deploy:** trust lists carry {HOME} and render per machine, in the form each tool is known to accept ([#1373](https://github.com/mlorentedev/dotfiles/issues/1373)) ([470fbd7](https://github.com/mlorentedev/dotfiles/commit/470fbd797a826a4bb3ee6aab99793d70c4fcdaa5)), closes [#1334](https://github.com/mlorentedev/dotfiles/issues/1334)
* **env:** dotf env persist records what it wrote and sweeps a name the contract retired, bounded to that record ([#1378](https://github.com/mlorentedev/dotfiles/issues/1378)) ([8de1cd1](https://github.com/mlorentedev/dotfiles/commit/8de1cd19962797cf3ba2164f5f43eec37800cef6))
* **env:** dotf env persist writes the contract variables where a profile-less process reads them ([#1362](https://github.com/mlorentedev/dotfiles/issues/1362)) ([389aaf5](https://github.com/mlorentedev/dotfiles/commit/389aaf50f3d0246b56b1e27871094694dbf44701))
* **harness:** dotf harness mirror, one deploy-dir mirror for both OSes ([#1305](https://github.com/mlorentedev/dotfiles/issues/1305)) ([acf78b3](https://github.com/mlorentedev/dotfiles/commit/acf78b3cc37633aeac0c9548ac32944fbe7459e5))
* **harness:** dotf harness presence injects the persona roster on every OS, so Windows finally gets it ([#1368](https://github.com/mlorentedev/dotfiles/issues/1368)) ([c4c43d6](https://github.com/mlorentedev/dotfiles/commit/c4c43d61522f6124ddbd4451d0870addae13bdfb))
* **model-map:** copilot gets tier entries measured against the seat, and leaves tierlessRenderers ([#1374](https://github.com/mlorentedev/dotfiles/issues/1374)) ([0f2ab42](https://github.com/mlorentedev/dotfiles/commit/0f2ab42096d5d54d9635dca572a50836a626a3e8))
* **orca:** dotf orca tune-hooks is the DX-006 repair, and the two shell twins that carried it are gone ([#1384](https://github.com/mlorentedev/dotfiles/issues/1384)) ([4718e46](https://github.com/mlorentedev/dotfiles/commit/4718e4685d3c4aab4dcbff6a400581f5c36b961f))
* **profile:** agyp, the PowerShell twin of the zsh saved-prompt launcher ([#1355](https://github.com/mlorentedev/dotfiles/issues/1355)) ([f373597](https://github.com/mlorentedev/dotfiles/commit/f3735975d68adbff244bc2631dfea4f683d4f745))
* **secrets:** the bw serve daemon leaves a log and a pid file, and doctor reads them when it dies ([#1348](https://github.com/mlorentedev/dotfiles/issues/1348)) ([097318e](https://github.com/mlorentedev/dotfiles/commit/097318ea4f468ed0848ba03546d9887537d5515a)), closes [#1315](https://github.com/mlorentedev/dotfiles/issues/1315)
* **setup:** Windows installs Claude Code and agy through their official installers, as Linux has since ADR-009 ([#1360](https://github.com/mlorentedev/dotfiles/issues/1360)) ([6e2dcd2](https://github.com/mlorentedev/dotfiles/commit/6e2dcd29fbc12d4b8f32ab6c415baceb802bbcc0))
* **spec:** dotf spec review draws one reviewer at random from a pool of five ([#1372](https://github.com/mlorentedev/dotfiles/issues/1372)) ([acca2da](https://github.com/mlorentedev/dotfiles/commit/acca2dabf3ff636eb0ea88be846166bf06c7c3fa)), closes [#1370](https://github.com/mlorentedev/dotfiles/issues/1370)
* **tools:** copilot joins the npm catalog on every OS, and doctor reports it against the pin ([#1359](https://github.com/mlorentedev/dotfiles/issues/1359)) ([f18b69a](https://github.com/mlorentedev/dotfiles/commit/f18b69a91c236497f7b617ea434db291ea40ac52))
* **tools:** one install channel per tool class, opencode through packages.json, dotf tools version ([#1311](https://github.com/mlorentedev/dotfiles/issues/1311)) ([016bf1a](https://github.com/mlorentedev/dotfiles/commit/016bf1a8959c85e83ebc56def8471382fe3d4baf))


### Bug Fixes

* **cli:** Windows deploy defects measured on a real box ([#1304](https://github.com/mlorentedev/dotfiles/issues/1304)) ([8680e56](https://github.com/mlorentedev/dotfiles/commit/8680e568b45c38f6287454933c55ff6202cb35f9))
* **copilot:** config keys the CLI reads, Co-authored-by off, dead exports removed ([#1307](https://github.com/mlorentedev/dotfiles/issues/1307)) ([431220c](https://github.com/mlorentedev/dotfiles/commit/431220c87c6cca4f997fc3c46e5a17626c102e8f))
* **deploy:** the manifest is version 2, and a dotf that cannot fully read it refuses it ([#1369](https://github.com/mlorentedev/dotfiles/issues/1369)) ([2c165fc](https://github.com/mlorentedev/dotfiles/commit/2c165fc8f9498013a71867b2b75c56f4d78e3fa5))
* **doctor:** declare the git-for-windows floor that carries the hooksPath fix ([#1350](https://github.com/mlorentedev/dotfiles/issues/1350)) ([9ec3c60](https://github.com/mlorentedev/dotfiles/commit/9ec3c60d476ecce95ba378a4c0e5ea08570717b8))
* **doctor:** declare the runner's missing Bitwarden identity instead of allow-listing its FAIL ([#1346](https://github.com/mlorentedev/dotfiles/issues/1346)) ([3b1f1f9](https://github.com/mlorentedev/dotfiles/commit/3b1f1f9a848af21d2daa6aeb860423aa2736dfe9)), closes [#1313](https://github.com/mlorentedev/dotfiles/issues/1313)
* **doctor:** detect a BUG-020-corrupted PowerShell profile and heal it under --fix ([#1353](https://github.com/mlorentedev/dotfiles/issues/1353)) ([5078d49](https://github.com/mlorentedev/dotfiles/commit/5078d490d9ac2b89d0ea3aa2e21058eebcfdc44c))
* **doctor:** the profile doctor measures is the one pwsh names, and the heal is told which file ([#1379](https://github.com/mlorentedev/dotfiles/issues/1379)) ([dab595e](https://github.com/mlorentedev/dotfiles/commit/dab595e34a72d6c304fb3335061ecda0a26297fa))
* **harness:** read both skills: frontmatter forms when rendering presence ([#1319](https://github.com/mlorentedev/dotfiles/issues/1319)) ([4e01907](https://github.com/mlorentedev/dotfiles/commit/4e01907bdec69a508e31e194b13ba734b543c686))
* **secrets:** unlock syncs the daemon's vault cache, and doctor reports that cache's age ([#1352](https://github.com/mlorentedev/dotfiles/issues/1352)) ([03a47ac](https://github.com/mlorentedev/dotfiles/commit/03a47ac707156fe8dc15d295b59b5abc7c7f1e72))
* **setup-windows:** the skill renderer drops neutral frontmatter, so Claude Code discovers every skill again ([#1386](https://github.com/mlorentedev/dotfiles/issues/1386)) ([cb5d12c](https://github.com/mlorentedev/dotfiles/commit/cb5d12c898f233199d09f31a8ec3acb53d8df199))
* **setup:** bare dotf deploy so every declared config is installed by both setups ([#1351](https://github.com/mlorentedev/dotfiles/issues/1351)) ([7a135a0](https://github.com/mlorentedev/dotfiles/commit/7a135a06b78d8f7c070beedacaf9ac8498f6d1f9)), closes [#1301](https://github.com/mlorentedev/dotfiles/issues/1301)
* **setup:** scripts deploy to the contract's directory, and the retired ones are swept ([#1356](https://github.com/mlorentedev/dotfiles/issues/1356)) ([2d87c7b](https://github.com/mlorentedev/dotfiles/commit/2d87c7bd05fcc0beca7af5e8455ba4de25fd614d))
* **setup:** UTF-8 console, LF deployed .md files, recursive sensitive/ mirror on Windows ([#1306](https://github.com/mlorentedev/dotfiles/issues/1306)) ([dbb9c60](https://github.com/mlorentedev/dotfiles/commit/dbb9c6066815d354cef145a727b00d6f4c722814))
* **setup:** Windows installs uv before registering the Claude MCP servers, so a first run registers hive and pdf-modifier ([#1367](https://github.com/mlorentedev/dotfiles/issues/1367)) ([2802434](https://github.com/mlorentedev/dotfiles/commit/2802434968470a7f2df61687593aff253b1615b6))
* **shell:** the agent-wrapper guard ran before the PATH that satisfies it ([#1283](https://github.com/mlorentedev/dotfiles/issues/1283)) ([5363a2d](https://github.com/mlorentedev/dotfiles/commit/5363a2d8685f6ad57360239f49a6f603fb4c9d30))
* **spec:** the drawn reviewer is told who it is, and a foreground run that wrote nothing fails ([#1388](https://github.com/mlorentedev/dotfiles/issues/1388)) ([03e7bb8](https://github.com/mlorentedev/dotfiles/commit/03e7bb86f03bf2b349f3d816ae3d8d6246646e76)), closes [#1383](https://github.com/mlorentedev/dotfiles/issues/1383)

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

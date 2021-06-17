package audit

func newClusterSuite(r *Run) *Suite {
	suite := &Suite{
		Name:          "cluster",
		Enable:        true,
		Configuration: r.suiteConfig("cluster"),
		Core:          true,
		run:           r,
		log:           r,
		checks:        make(map[string]*Check),
	}

	newServerVersionsCheck(suite)

	return suite
}

func newServerVersionsCheck(s *Suite) *Check {
	check := &Check{
		Name:      "server_versions",
		Version:   version,
		Suite:     "cluster",
		Enable:    true,
		Core:      true,
		Clustered: true,
		clusterCheck: func(servers []*Server, result *CheckResult, log Logger) error {
			result.Outcome = UnknownOutcome

			result.Assert(func() {
				if len(servers) < 1 {
					log.Infof("Skipping %s on non clustered servers", result.CheckName)
					result.Skipped = true
				}
			})

			meta := map[string][]string{
				"ok":     {},
				"failed": {},
			}
			result.Metadata = meta

			result.Assert(func() {
				desired := servers[0].Info.Version
				log.Infof("Comparing %d server versions against version %s", len(servers), desired)

				for _, srv := range servers[1:] {
					if srv.Info.Version != desired {
						result.Outcome = ErrorOutcome
						result.ErrorF("Version %s does not match %s on %s", srv.Info.Version, desired, srv)
						meta["failed"] = append(meta["failed"], srv.Info.ID)
					} else {
						meta["ok"] = append(meta["ok"], srv.Info.ID)
					}
				}

				if result.Outcome != ErrorOutcome {
					result.Outcome = OKOutcome
				}
			})

			return nil
		},
	}

	s.registerCheck(check)

	return check
}

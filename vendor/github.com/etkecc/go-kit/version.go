package kit

import "runtime/debug"

// Version tries override, Main.Version, Deps[module], VCS revision(+"-dirty"), then fallback, first hit wins.
func Version(module, fallback string, override ...string) string {
	if len(override) > 0 && override[0] != "" {
		return override[0]
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallback
	}

	var v string
	if module == "" || info.Main.Path == module {
		v = mainVersion(info)
	} else {
		v = depVersion(info, module)
	}
	if v != "" {
		return v
	}
	return fallback
}

// mainVersion is the app's own version: the stamped release, else the git revision it was built from.
func mainVersion(info *debug.BuildInfo) string {
	if v := cleanVersion(info.Main.Version); v != "" {
		return v
	}
	return vcsVersion(info)
}

// depVersion follows any replace directive for module's version, since dep.Version alone is the stale pre-replace pin.
func depVersion(info *debug.BuildInfo, module string) string {
	for _, dep := range info.Deps {
		if dep.Path != module {
			continue
		}
		if dep.Replace != nil {
			dep = dep.Replace
		}
		return cleanVersion(dep.Version)
	}
	return ""
}

// UserAgent builds a "product/version" string via Version with a v0.0.0 fallback; module is the caller's own path.
func UserAgent(product, module string, override ...string) string {
	return product + "/" + Version(module, "v0.0.0", override...)
}

// cleanVersion drops the "(devel)" placeholder so an unstamped build falls through to the next source.
func cleanVersion(v string) string {
	if v == "(devel)" {
		return ""
	}
	return v
}

// vcsVersion is the git "revision(-dirty)" the binary was built from, for when no release version was stamped.
func vcsVersion(info *debug.BuildInfo) string {
	var rev, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev == "" {
		return ""
	}
	return rev + dirty
}

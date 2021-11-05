package entity

func getUpstreamDef(obj interface{}) (uf *UpstreamDef) {
	switch obj := obj.(type) {
	case *Upstream:
		uf = &obj.UpstreamDef
	case *Service:
		uf = obj.Upstream
	case *Route:
		uf = obj.Upstream
	default:
		return nil
	}
	return
}

func ServiceFilter(obj interface{}) bool {
	uf := getUpstreamDef(obj)
	if uf == nil {
		return false
	}

	if uf.ServiceName != "" && uf.DiscoveryType != "" {
		return true
	}
	return false
}

func ServiceUpdate(obj, newObj interface{}) bool {
	uf := getUpstreamDef(obj)
	newUf := getUpstreamDef(newObj)

	if uf.ServiceName != newUf.ServiceName || uf.DiscoveryType != newUf.DiscoveryType {
		return false
	}

	// Two pointers are equal only when they are both nil
	if uf.DiscoveryArgs != newUf.DiscoveryArgs &&
		(uf.DiscoveryArgs == nil || newUf.DiscoveryArgs == nil ||
			uf.DiscoveryArgs.GroupName != newUf.DiscoveryArgs.GroupName ||
			uf.DiscoveryArgs.NamespaceID != newUf.DiscoveryArgs.NamespaceID) {
		return true
	}

	return false
}

func ServiceReplace(obj, newObj interface{}) bool {
	uf := getUpstreamDef(obj)
	newUf := getUpstreamDef(newObj)

	return uf.ServiceName != newUf.ServiceName || uf.DiscoveryType != newUf.DiscoveryType
}

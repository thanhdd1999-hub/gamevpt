package pet

func (p *Pet) EvolutionInfo() map[string]interface{} {
	result := map[string]interface{}{"envo": false, "round": 0, "stage": 0}
	if p == nil {
		return result
	}
	if info, ok := p.Property["envoInfo"].(map[string]interface{}); ok {
		for key, value := range info {
			result[key] = value
		}
	}
	return result
}

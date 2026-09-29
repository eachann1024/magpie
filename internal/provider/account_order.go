package provider

import "fmt"

// SetAccountOrder saves only presentation: credentials, enabled accounts and
// routing remain untouched. Reject stale lists rather than losing an account
// added or removed while the editor was open.
func SetAccountOrder(id string, order []string) error {
	p, err := Find(id)
	if err != nil {
		return err
	}
	var available []string
	if p.Account != nil {
		for _, l := range Logins(p.Account.Agent) {
			available = append(available, l.User)
		}
	} else {
		for _, k := range p.KeyList() {
			available = append(available, k.ID)
		}
	}
	if err := validateAccountOrder(available, order); err != nil {
		return err
	}
	p.AccountOrder = append([]string{}, order...)
	return Save(*p)
}

func validateAccountOrder(available, order []string) error {
	if len(available) != len(order) || len(order) == 0 {
		return fmt.Errorf("accounts changed; reopen the provider and try again")
	}
	ids := make(map[string]bool, len(available))
	for _, id := range available {
		ids[id] = true
	}
	for _, id := range order {
		if !ids[id] {
			return fmt.Errorf("unknown or repeated account; reopen the provider and try again")
		}
		delete(ids, id)
	}
	return nil
}

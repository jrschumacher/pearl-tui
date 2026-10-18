// Package form asks for a name with a Huh form nobody themed: C-HUHTHEME.
package form

import "charm.land/huh/v2"

// Ask runs the form in Huh's default colours.
func Ask() (string, error) {
	var name string
	err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Name").Value(&name))).Run()
	return name, err
}

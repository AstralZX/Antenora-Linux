// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import "fmt"

// Toolchain bootstraps the Antenora build toolchain by installing the
// toolchain meta-package (binutils, gcc, make, autotools, pkg-config, headers).
func (d *Dante) Toolchain() error {
	fmt.Println(":: Bootstrapping the Antenora toolchain (this may take a while)...")
	return d.Install("toolchain", false)
}

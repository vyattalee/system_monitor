package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	str "strings"

	"github.com/atotto/clipboard"
	"github.com/dustin/go-humanize"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var (
	version                   = "2.4"            /* Version variable */
	termGrid, dfGrid, pkgGrid *ui.Grid           /* Grid widgets for the layout */
	pkgText, sysInfoText      *widgets.Paragraph /* Paragraph widgets for showing text */
	cmdList                   *widgets.List      /* List widget for the executed commands. */
	sortPackages              = false            /* Boolean value for sorting the packages alphabetically */
	reversePackages           = false            /* Boolean value for reversing the package list. */
	dfIndex, pkgIndex         = 0, 0             /* Index value for the disk usage widgets & package list */
	showInfo                  = false            /* Switch to the package information page */
	pkgMode                   = 0                /* Integer value for changing the package operation mode. */
	pkgModes                  = []string{        /* Package management/operation modes */
		"remove", "install", "upgrade", "go-to", "search",
	}
	pacmanBackend           = "pacman"
	termColor               = "blue"                  /* Default color of the dashboard. */
	inputQuery, inputSuffix = "", ""                  /* List title suffix & input query value */
	cmdPrefix               = " λ ~ "                 /* Prefix for prepending to the commands */
	cmdConfirm              = " [y] "                 /* Confirmation string for commands to execute */
	osIDCmd                 = "awk -F '=' '/^ID=/ " + /* Print the OS ID information (for distro checking) */
		"{print tolower($2)}' /etc/*-release 2>/dev/null"
	sysInfoCmd = "printf \"Hostname: $(uname -n)\\n" + /* Print the system information with 'uname' */
		" Kernel: $(uname -s)\\n" +
		" Kernel Release: $(uname -r)\\n" +
		" Kernel Version: $(uname -v)\\n" +
		" Processor Type: $(uname -p)\\n" +
		" Hardware: $(uname --m)\\n" +
		" Hardware Platform: $(uname -i)\\n" +
		" OS: $(uname -o)\\n\""
	dfCmd   = "df -h 2> /dev/null | awk '{$1=$1};1 {if(NR>1)print}'" /* Print the disk usage with 'df' */
	pkgsCmd = map[string]string{                                     /* Commands for listing the installed packages */
		"arch,manjaro": "pacman -Qi | awk '/^Name/{name=$3} " +
			"/^Version/{ver=$3} " +
			"/^Description/{desc=substr($0,index($0,$3))} " +
			"/^Installed Size/{size=$4$5; " +
			"print name \";\" ver \";\" size \";\" desc}' " +
			"| sort -h -r -t ';' -k3 " +
			"&& echo \"pacman -Qi %s | sed -e 's/^/  /';" +
			"pacman -Rcns %s --noconfirm;pacman -S %s --noconfirm;" +
			"pacman -Syu %s --noconfirm;x\"" +
			"&& echo 'Name|Version|Installed Size|Description'",
		"debian,ubuntu,mint": "dpkg-query -W --showformat='${Package};${Version};" +
			"${Installed-Size};${binary:Summary}\\n' | sort -n -r -t ';' -k3 " +
			"&& echo \"apt-cache show %s | sed -e 's/^/  /';apt-get -y remove %s;" +
			"apt-get -y install %s;apt-get -y install --only-upgrade %s;x\" " +
			"&& echo 'Name|Version|Installed Size|Description'",
		"suse": "rpm -qa --queryformat '%{Name};%{Version};%{Size};%{Summary}\\n' | " +
			"sort -n -r -t ';' -k3  && echo \"rpm -qi %s | sed -e 's/^/  /';" +
			"zypper rm -y %s;zypper in -y %s;zypper up -y %s;x\" && " +
			"echo 'Name|Version|Installed Size|Description'",
		"fedora,centos,redhat": "rpm -qa --queryformat " +
			"'%{Name};%{Version};%{Size};%{Summary}\\n' | " +
			"sort -n -r -t ';' -k3  && echo \"rpm -qi %s | sed -e 's/^/  /';" +
			"dnf -y remove %s;dnf -y install %s;dnf -y upgrade %s;x\" && " +
			"echo 'Name|Version|Installed Size|Description'",
		"void": "xbps-query -l | awk '{print $2 \";\" substr($0,index($0,$3))}' " +
			"&& echo \"xbps-query %s | sed -e 's/^/  /';xbps-remove -y %s;" +
			"xbps-install -Sy %s;xbps-install -Sy %s\" && echo 'Name-version|Description'",
		"gentoo": "EIX_LIMIT_COMPACT=0 eix -c | awk '{name=$2} {ver=$3; gsub(/\\:|\\(|\\)/, \"\", ver)} " +
			"!/Found|\\[1]|^$/{desc=substr($0,index($0,$4)); print name \";\" ver \";\" desc}' " +
			"&& echo \"equery m %s | sed -e 's/^/  /';emerge -cv %s;emerge %s;emerge -uDU %s\" " +
			"&& echo 'Name|Version|Description'",
		"nixos": "nix-env -qa --description | sed -e 's/  \\+/;/g' && echo \"nix-env -qa --meta --xml %s " +
			"| grep -o '\\\".*\\\"' | sed -e 's/\\\"//g' -e 's/ type=[a-z]*//g' -e 's/ value=/: /g' " +
			"-e 's/^/  /';nix-env -e %s;nix-env -i %s;nix-env -u %s\" && echo 'Name-version|Description'",
		"guix": "guix package --list-available | sed -e 's/  \\+/;/g; s/\\t/;/g' && " +
			"echo \"guix package --show=%s | sed -e 's/^/  /';guix package -r %s;guix package -i %s;" +
			"guix package -u %s;x\" && echo 'Name|Version|Output|Location'",
	}
	keyActions = "   Key                     Action\n" +
		"   ?                       : Help\n" +
		"   enter, space, tab       : Show package information\n" +
		"   i                       : Install package\n" +
		"   u/ctrl-u                : Upgrade package/with input\n" +
		"   r/ctrl-r                : Remove package/with input\n" +
		"   s,/                     : Search package\n" +
		"   g                       : Go to package (index)\n" +
		"   y                       : Confirm and execute the selected command\n" +
		"   p                       : Copy selected package name/information\n" +
		"   e                       : Copy selected command\n" +
		"   c                       : Scroll executed commands list\n" +
		"   j/k, down/up            : Scroll down/up (packages)\n" +
		"   ctrl-j/ctrl-k           : Scroll to bottom/top (packages)\n" +
		"   l/h, right/left         : Scroll down/up (disk usage)\n" +
		"   backspace               : Go back\n" +
		"   q, esc, ctrl-c, ctrl-d  : Exit\n"
)

/*!
 * Parse the 'df' command output as Gauge and GridItem.
 *
 * \param diskUsage (array of 'df' lines)
 * \param s (starting index)
 * \param n (n * entry)
 * \return gauges, entries
 */
func getDfEntries(diskUsage []string, s int, n int) ([]*widgets.Gauge,
	[]interface{}) {
	/* Use the length of 'df' array if "n"
	 * (entry count to show) is greater.
	 */
	if len(diskUsage) < n {
		n = len(diskUsage)
	}
	entries := make([]interface{}, n)
	var gauges []*widgets.Gauge
	for i := s; i < s+n; i++ {
		/* Pass the insufficient lines. */
		if len(diskUsage[i]) < 5 {
			continue
		}
		/* Create gauge widget from the splitted
		 * line and add it to the entries slice.
		 */
		dfVal := str.Split(diskUsage[i], " ")
		dfGau := widgets.NewGauge()
		dfGau.Title = fmt.Sprintf("%s ~ (%s/%s) [%s]",
			dfVal[0], dfVal[2], dfVal[1], dfVal[len(dfVal)-1])
		dfGau.BorderStyle.Fg = ui.ColorBlack
		percent, err := strconv.Atoi(
			str.Replace(dfVal[4], "%", "", 1))
		if err != nil {
			return gauges, nil
		}
		dfGau.Percent = percent
		if percent > 95 {
			dfGau.BarColor = ui.ColorRed
		}
		gauges = append(gauges, dfGau)
		entries[i-s] = ui.NewRow(
			1.0/float64(n),
			ui.NewCol(1.0, dfGau),
		)
	}
	return gauges, entries
}

/*!
 * Execute the 'df' command and show parsed output values with widgets.
 *
 * \param dfIndex (starting index of entries to render)
 * \return dfIndex
 */
func showDfInfo(dfIndex int) int {
	/* Prevent underflow and return the first index. */
	if dfIndex < 0 {
		return 0
	}
	/* Find the optimal widget count for the Grid. */
	dfCount := (sysInfoText.Block.Inner.Max.Y + 1) / 3
	/* Execute the 'df' command and split the output by newline. */
	dfOutput := str.Split(execCmd("sh", "-c", dfCmd), "\n")
	/* Return the maximum index on overflow. */
	if len(dfOutput)-dfIndex < dfCount && len(dfOutput) > dfCount {
		return len(dfOutput) - dfCount
		/* Use the first index on invalid entry count. */
	} else if len(dfOutput) <= dfCount {
		dfIndex = 0
	}
	/* Create and render the widgets. */
	gauges, dfEntries := getDfEntries(
		dfOutput,
		dfIndex,
		dfCount)
	dfGrid.Set(dfEntries...)
	ui.Render(dfGrid)
	for _, g := range gauges {
		ui.Render(g)
	}
	return dfIndex
}

/*!
 * Parse the 'packages' command output as List widgets (GridItem) for Grid.
 *
 * \param pkgs (output lines)
 * \return pkgls, entries, optCmds
 */
func getPkgListEntries(pkgs []string) ([]*widgets.List,
	[]interface{}, []string) {
	/* Create a slice of List widgets. */
	var pkgls []*widgets.List
	/* Create the title and option command slices from the last lines. */
	titles, optCmds := str.Split(pkgs[len(pkgs)-1], "|"),
		str.Split(pkgs[len(pkgs)-2], ";")
	/* Loop through the lines for creating GridItems that contain List widget. */
	entries := make([]interface{}, len(titles))
	for i := 0; i < len(titles); i++ {
		/* Parse the line for package details and append to the 'rows'. */
		var rows []string
		for _, pkg := range pkgs {
			/* Pass the lines that have insufficient length. */
			if len(str.Split(pkg, ";")) != len(titles) {
				continue
			}
			/* Convert size to human readable format if possible. */
			if size, err := strconv.ParseInt(str.Split(pkg, ";")[i], 10, 64); err == nil && titles[i] == "Installed Size" {
				rows = append(rows, " "+humanize.Bytes(uint64(size)))
			} else {
				rows = append(rows, " "+str.Split(pkg, ";")[i])
			}
		}
		/* Create a List widget and initialize with the parsed values. */
		pkgl := widgets.NewList()
		pkgl.Title = titles[i]
		pkgl.Rows = rows
		pkgl.WrapText = false
		pkgl.Border = false
		pkgl.TextStyle = ui.NewStyle(ui.StyleParserColorMap[termColor])
		/* Add List widget to the GridItem slice. */
		if i == 0 {
			entries[i] = ui.NewCol(1.0/float64(len(titles)), pkgl)
		} else if i == len(titles)-1 {
			entries[i] = ui.NewCol(1.0, pkgl)
		} else {
			entries[i] = ui.NewCol(1.0/(float64(len(titles))*1.6), pkgl)
		}
		pkgls = append(pkgls, pkgl)
	}
	return pkgls, entries, optCmds
}

/*!
 * Scroll and render a slice of List widgets.
 *
 * \param lists
 * \param amount
 * \param row
 * \param force
 * \return 0 on success
 */
func scrollLists(lists []*widgets.List, amount int,
	row int, force bool) int {
	for i, l := range lists {
		if row != -1 && row <= len(l.Rows) {
			l.SelectedRow = row
		} else {
			l.ScrollAmount(amount)
		}
		if i == len(lists)-1 && l.Title != "" {
			l.Title = fmt.Sprintf("%s (%d/%d)",
				str.Split(l.Title, " (")[0],
				l.SelectedRow+1, len(l.Rows))
		}
		if len(l.Rows) != 0 || force {
			ui.Render(l)
		}
	}
	return 0
}

/*!
 * Execute a operating system command and capture the output.
 *
 * \param name
 * \param arg
 * \return output
 */
func execCmd(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Execution of '%s' failed with %s\n", name, err)
	}
	return str.TrimSpace(string(out))
}

/*!
 * Initialize, execute, render and handle.
 *
 * \param osID (Operating system identity)
 * \return 0 on exit
 */
func start(osID string) int {
	/* Initialize the termui library. */
	if err := ui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui: %v", err)
	}
	/* Close the UI on function exit. */
	defer ui.Close()
	/* Initialize the widgets. */
	termGrid, dfGrid, pkgGrid =
		ui.NewGrid(),
		ui.NewGrid(),
		ui.NewGrid()
	cmdList = widgets.NewList()
	cmdList.WrapText = false
	cmdList.TextStyle = ui.NewStyle(ui.StyleParserColorMap[termColor])
	cmdList.BorderStyle.Fg = ui.ColorBlack
	pkgText, sysInfoText =
		widgets.NewParagraph(),
		widgets.NewParagraph()
	pkgText.WrapText = true
	pkgText.Text = "" +
		"       [.smNNho:\n" +
		"     ..   -+hmMmh+\n" +
		"   -smNds:.  `sMMN\n" +
		"    `-+hNMNs  +MMN\n" +
		"     .  oMMd  /MMN\n" +
		"  .pkg` +MMd  /MMd\n" +
		"  `top` omh/  -o:`](fg:white,mod:bold)\n" +
		" > [github.com/orhun/pkgtop](fg:" + termColor + ")\n" +
		" > [Interactive package manager & resource monitor (v" +
		version + ")](fg:" + termColor + ")"
	pkgText.BorderStyle.Fg = ui.ColorBlack
	sysInfoText.BorderStyle.Fg = ui.ColorBlack
	/* Set the operating system variable. */
	osID = str.ToLower(str.TrimSpace(str.Split(osID, "\n")[0]))
OSCheckLoop:
	for ids := range pkgsCmd {
		for _, id := range str.Split(ids, ",") {
			if str.Contains(osID, id) {
				osID = ids
				break OSCheckLoop
			}
		}
	}
	if str.Contains(osID, "arch") {
		pkgsCmd[osID] = str.ReplaceAll(pkgsCmd[osID], "pacman", pacmanBackend)
	}
	/* Update the commands list. */
	cmdList.Rows = []string{cmdPrefix + pkgsCmd[osID],
		cmdPrefix + osIDCmd}
	/* Retrieve packages with the OS command. */
	pkgs := str.Split(execCmd("env", "LC_ALL=C", "sh", "-c", pkgsCmd[osID]), "\n")
	/* Check if the operating system command exists. */
	if _, hasKey := pkgsCmd[osID]; !hasKey {
		ui.Close()
		keys := make([]string, 0, len(pkgsCmd))
		for k := range pkgsCmd {
			keys = append(keys, k)
		}
		log.Fatalf("Failed to start pkgtop on '%s'. "+
			"Try providing Linux distribution with -d argument. (%s)",
			osID, str.Join(keys, ","))
		/* Check the packages count. */
	} else if len(pkgs) < 2 {
		ui.Close()
		log.Fatalf("Failed to retrieve package list on '%s'.", osID)
	}
	/* Sort package names if the command line argument provided. */
	if sortPackages {
		sort.Strings(pkgs[:len(pkgs)-2])
	}
	/* Reverse the package list if the command line argument provided. */
	if reversePackages {
		for i := len(pkgs)/2 - 1; i >= 0; i-- {
			opp := len(pkgs) - 3 - i
			pkgs[i], pkgs[opp] = pkgs[opp], pkgs[i]
		}
	}
	/* Initialize and render the widgets for showing the package list. */
	lists, pkgEntries, optCmds := getPkgListEntries(pkgs)
	scrollLists(lists, -1, 0, false)
	pkgGrid.Set(ui.NewRow(1.0, pkgEntries...))
	/* Show the OS information. */
	cmdList.Rows = append([]string{cmdPrefix + sysInfoCmd}, cmdList.Rows...)
	for _, info := range str.Split(" "+execCmd("sh", "-c", sysInfoCmd), "\n") {
		sysInfoText.Text += "[" + str.Split(info, ":")[0] + ":](fg:" + termColor + ")" +
			str.Join(str.Split(info, ":")[1:], "") + "\n"
	}
	/* Configure and render the main grid layout.
	* ...................................................
	* :  [Disk Usage]  : [System Info] : [Project Info] :
	* :................:...............:................:
	* :                                                 :
	* :               [Installed Packages]              :
	* :                                                 :
	* :.................................................:
	* :                   [Commands]                    :
	* :.................................................:
	 */
	termWidth, termHeight := ui.TerminalDimensions()
	termGrid.SetRect(0, 0, termWidth, termHeight)
	termGrid.Set(
		ui.NewRow(1.0/4,
			ui.NewCol(1.0/2, dfGrid),
			ui.NewCol(0.3, sysInfoText),
			ui.NewCol(1.0/5, pkgText),
		),
		ui.NewRow(1.0/1.6,
			ui.NewCol(1.0/1, pkgGrid),
		),
		ui.NewRow(1.0/8,
			ui.NewCol(1.0, cmdList),
		),
	)
	ui.Render(pkgGrid, termGrid)
	/* Show the disk usage information. (post-render) */
	dfIndex = showDfInfo(dfIndex)

	/* Get events from termui. */
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			/* Allow typing to the search area if the search mode is on. */
			if pkgMode != 0 && (len(str.ToLower(e.ID)) == 1 ||
				str.ToLower(e.ID) == "<backspace>") {
				/* Delete the last char from query on the backspace key press. */
				if len(inputQuery) != 0 && str.ToLower(e.ID) == "<backspace>" {
					inputQuery = inputQuery[:len(inputQuery)-1]
					/* Append key to the query. */
				} else if str.ToLower(e.ID) != "<backspace>" {
					/* Check for 'go-to' mode. */
					if pkgMode != 4 {
						inputQuery += str.ToLower(e.ID)
						/* Allow typing only numbers for the package index. */
					} else if _, err := strconv.Atoi(e.ID); err == nil {
						inputQuery += e.ID
					}
				}
				if pkgMode == 5 {
					/* Create lists again for searching. */
					searchLists, _, _ := getPkgListEntries(pkgs)
					/* Empty the current list rows. */
					for _, l := range lists {
						l.Rows = nil
					}
					/* Loop through the first list, compare the query and show results. */
					for s, name := range searchLists[0].Rows {
						if str.Contains(name, inputQuery) {
							for i, l := range searchLists {
								lists[i].Rows = append(lists[i].Rows, l.Rows[s])
							}
						}
					}
				}
				/* Update the search area. */
				lists[0].Title = inputSuffix + inputQuery
				/* Scroll and (force) render the lists. */
				scrollLists(lists, -1, 0, true)
				break
			}
			switch str.ToLower(e.ID) {
			/* Exit search mode or quit. */
			case "q", "<escape>", "<c-c>", "<c-d>":
				if pkgMode == 0 {
					return 0
				}
				pkgMode = 0
			/* Terminal resize. */
			case "<resize>":
				payload := e.Payload.(ui.Resize)
				termGrid.SetRect(0, 0,
					payload.Width, payload.Height)
				ui.Clear()
				ui.Render(termGrid)
				dfIndex = showDfInfo(dfIndex)
				scrollLists(lists, -1, lists[0].SelectedRow, false)
			/* Scroll down. (packages) */
			case "j", "<down>", "<pagedown>":
				scrollLists(lists, 1, -1, false)
			/* Scroll to bottom. (packages) */
			case "<c-j>":
				scrollLists(lists, -1,
					len(lists[0].Rows)-1, false)
			/* Scroll up. (packages) */
			case "k", "<up>", "<pageup>":
				scrollLists(lists, -1, -1, false)
			/* Scroll to top. (packages) */
			case "<c-k>":
				scrollLists(lists, -1, 0, false)
			/* Scroll down. (disk usage) */
			case "l", "<right>":
				dfIndex = showDfInfo(dfIndex + 1)
			/* Scroll up. (disk usage) */
			case "h", "<left>":
				dfIndex = showDfInfo(dfIndex - 1)
			/* Scroll executed commands list. */
			case "c":
				if cmdList.SelectedRow < len(cmdList.Rows)-1 {
					cmdList.ScrollDown()
				} else {
					cmdList.ScrollTop()
				}
				ui.Render(cmdList)
			/* Copy selected package info to clipboard. */
			case "p":
				if lists[0] != nil && len(lists[0].Rows) != 0 &&
					lists[0].SelectedRow >= 0 {
					clipboard.WriteAll(str.TrimSpace(
						lists[0].Rows[lists[0].SelectedRow]))
				}
			/* Copy executed command to clipboard. */
			case "e":
				if len(cmdList.Rows) != 0 && cmdList.SelectedRow >= 0 {
					cmdReplacer := str.NewReplacer(cmdPrefix, "", cmdConfirm, "")
					clipboard.WriteAll(str.TrimSpace(
						cmdReplacer.Replace(cmdList.Rows[cmdList.SelectedRow])))
				}
			/* Go back from information page. */
			case "<backspace>":
				showInfo = true
				fallthrough
			/* Show package information or help message. */
			case "<enter>", "<space>", "<tab>", "?":
				if !showInfo && len(lists[0].Rows) != 0 {
					/* Append operation command to list if any mode is on. */
					if pkgMode > 0 && pkgMode != 5 && inputQuery != "" {
						if pkgMode != 4 {
							pkgOptCmd := fmt.Sprintf(optCmds[pkgMode], inputQuery)
							if cmdList.Rows[0] != cmdConfirm+pkgOptCmd {
								cmdList.Rows = append([]string{cmdConfirm + pkgOptCmd},
									cmdList.Rows...)
							}
							cmdList.ScrollTop()
							ui.Render(cmdList)
							/* Scroll to the entered row. */
						} else {
							rowInput, _ := strconv.Atoi(inputQuery)
							scrollLists(lists, -1, rowInput-1, false)
						}
						pkgMode = 0
						break
					}
					/* Help message. */
					infoRow := keyActions
					/* Check pressed key for showing the package information. */
					if str.Contains(str.ToLower(e.ID), "<") {
						/* Parse the 'package info' command output after execution,
						 * use first list for showing the information.
						 */
						pkgIndex = lists[0].SelectedRow
						selectedPkg := str.TrimSpace(lists[0].Rows[pkgIndex])
						pkgInfoCmd := fmt.Sprintf(optCmds[0], selectedPkg)
						/* Update the commands list. */
						if str.Contains(cmdList.Rows[0], str.Split(optCmds[0], "%s")[0]) &&
							str.Contains(cmdList.Rows[0], str.Split(optCmds[0], "%s")[1]) {
							cmdList.Rows[0] = cmdPrefix + pkgInfoCmd
						} else {
							cmdList.Rows = append([]string{cmdPrefix + pkgInfoCmd},
								cmdList.Rows...)
						}
						cmdList.ScrollTop()
						infoRow = "  " + execCmd("sh", "-c", pkgInfoCmd)
					}
					/* Prepare the list widget. */
					lists = lists[:1]
					lists[0].Title = ""
					lists[0].WrapText = !showInfo
					lists[0].Rows = []string{infoRow}
					/* Set the Grid entries. */
					pkgEntries = nil
					pkgEntries = append(pkgEntries, ui.NewCol(1.0, lists[0]))
					pkgGrid.Set(ui.NewRow(1.0, pkgEntries...))
					/* Disable the mode and set index for scrolling to the first row. */
					if pkgMode != 0 {
						pkgMode, pkgIndex = 0, 0
					}
				} else {
					/* Parse the packages with previous command output and show. */
					lists[0].Rows = nil
					lists[0].WrapText = false
					lists, pkgEntries, optCmds = getPkgListEntries(pkgs)
					pkgGrid.Set(ui.NewRow(1.0, pkgEntries...))
				}
				/* Set the flags for showing info and searching package. */
				showInfo = !showInfo
				ui.Render(pkgGrid, cmdList)
				scrollLists(lists, pkgIndex, -1, false)
			/* Search, install, upgrade or go-to package. */
			case "s", "i", "<c-u>", "g", "<c-r>", "/":
				/* Allow changing mode if not showing any package information. */
				if !showInfo {
					/* Set variables for switching the mode. */
					pressedKey := str.NewReplacer("<c-", "", ">", "", "/", "s").
						Replace(str.ToLower(e.ID))
					inputQuery = ""
					for i, v := range pkgModes {
						if v[:1] == pressedKey {
							pkgMode = i + 1
							/* Set the first lists title for the selected mode. */
							if str.Contains(inputSuffix, " > ") {
								inputSuffix = str.Split(inputSuffix, ">")[0] +
									"> " + v + ": "
							} else if !str.Contains(inputSuffix, v) {
								inputSuffix = lists[0].Title + " > " + v + ": "
							}
							break
						}
					}
					lists[0].Title = inputSuffix
					scrollLists(lists, -1, 0, false)
				}
			/* Remove or upgrade package. */
			case "r", "u":
				/* Break if no packages found to remove or showing information. */
				if len(lists[0].Rows) == 0 || showInfo {
					break
				}
				/* Set the command index (fixed value) depending on the pressed key. */
				optCmdIndex := 1
				if str.ToLower(e.ID) != "r" {
					optCmdIndex = 3
				}
				/* Add command to the command list with confirmation prefix. */
				selectedPkg := str.TrimSpace(lists[0].Rows[lists[0].SelectedRow])
				pkgOptCmd := fmt.Sprintf(optCmds[optCmdIndex], selectedPkg)
				if cmdList.Rows[0] != cmdConfirm+pkgOptCmd {
					cmdList.Rows = append([]string{cmdConfirm + pkgOptCmd},
						cmdList.Rows...)
				}
				cmdList.ScrollTop()
				ui.Render(cmdList)
			/* Confirm and execute the selected command. */
			case "y":
				selectedCmdRow := cmdList.Rows[cmdList.SelectedRow]
				if str.Contains(selectedCmdRow, cmdConfirm) {
					/* Close the UI, execute the command and show output. */
					ui.Close()
					cmd := exec.Command("sh", "-c",
						str.Replace(selectedCmdRow, cmdConfirm, "", -1))
					cmd.Stderr = os.Stderr
					cmd.Stdout = os.Stdout
					err := cmd.Run()
					/* Show the UI again if the execution is successful. */
					if err == nil {
						start(osID)
					}
				}
			}
		}
	}
}

/*!
 * Entry-point
 */
func main() {
	/* Parse command-line flags. */
	showVersion := flag.Bool("v", false, "print version")
	osID := flag.String("d", "", "linux distribution")
	flag.StringVar(&termColor, "c", "blue", "main color of the dashboard")
	flag.StringVar(&pacmanBackend, "pacman", "pacman", "select pacman backend (for Arch-based distro)")
	flag.BoolVar(&sortPackages, "a", false, "sort packages alphabetically")
	flag.BoolVar(&reversePackages, "r", false, "reverse the package list")
	flag.Parse()
	if *showVersion {
		fmt.Printf("pkgtop v%s\n", version)
		return
	} else if _, hasKey := ui.StyleParserColorMap[termColor]; !hasKey {
		fmt.Printf("color not supported.\n")
		return
	} else if *osID == "" {
		*osID = execCmd("sh", "-c", osIDCmd)
	}
	/* Initialize and start the termui. */
	start(*osID)
}

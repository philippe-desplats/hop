package i18n

// catalog holds every user-facing string. English is the fallback; any missing
// key in another language resolves to English, then to the key itself.
var catalog = map[Lang]map[string]string{
	EN: {
		"hub.hint.tab":     "↑↓ navigate · enter cd · tab actions · esc cancel",
		"hub.hint.shift":   "↑↓ navigate · enter cd · tab menu · esc cancel",
		"hub.hint.enter":   "↑↓ navigate · enter actions · esc cancel",
		"hub.shift_prefix": "shift → ",
		"hub.no_match":     "(no match)",
		"hub.git.none":     "no git repo",
		"hub.git.clean":    "clean",
		"hub.git.dirty":    "modified",
		"hub.actions.hint": "esc/tab back · ctrl+c cancel",

		"time.now":     "now",
		"time.fmt":     "%s ago",
		"time.u.min":   "min",
		"time.u.hour":  "h",
		"time.u.day":   "d",
		"time.u.week":  "w",
		"time.u.month": "mo",
		"time.u.year":  "y",

		"action.cd":          "cd here",
		"action.editor":      "open in editor",
		"action.ai":          "launch %s",
		"action.ai.resume":   "resume %s",
		"action.git":         "git status",
		"action.remote":      "open remote repo",
		"action.finder":      "open in Finder",
		"action.filemanager": "open in file manager",
		"action.mux":         "%s session",

		"action.short.resume":      "resume",
		"action.short.pin":         "pin",
		"action.short.unpin":       "unpin",
		"action.pin":               "pin to favorites",
		"action.unpin":             "remove from favorites",
		"action.short.remote":      "remote",
		"action.short.filemanager": "files",

		"config.title":          "hop · configuration",
		"config.field.command":  "Command name",
		"config.field.theme":    "Theme",
		"config.field.access":   "Hub actions access",
		"config.field.editor":   "Editor (open action)",
		"config.field.tmux":     "Show tmux action",
		"config.field.language": "Language",
		"config.field.roots":    "Scan roots",
		"config.field.depth":    "Scan depth",
		"config.field.ignore":   "Ignored folders",
		"config.help.command":   "the daily shortcut (p); takes effect on next exec zsh",
		"config.help.access":    "tab: Tab opens the menu · shift: UPPERCASE direct · enter: Enter opens the menu",
		"config.help.editor":    "command launched by the editor action",
		"config.help.roots":     "folders separated by spaces",
		"config.help.depth":     "levels below each root",
		"config.help.ignore":    "separated by spaces",
		"config.opt.no":         "no",
		"config.opt.yes":        "yes",
		"config.hint":           "↑↓ field · ←→ value · type to edit · enter/ctrl+s save · esc cancel",

		"cli.setup_hint":         "hop: no projects indexed yet, run `hop setup` to choose your project folders",
		"setup.title":            "hop · setup",
		"setup.roots.title":      "Which folders hold your projects?",
		"setup.roots.empty":      "No common project folder found under your home. You can add roots later with hop config.",
		"setup.roots.repos":      "%d repos",
		"setup.editor.title":     "Open projects in which editor?",
		"setup.editor.empty":     "No known editor found on PATH. You can set one later with hop config.",
		"setup.ai.title":         "Which AI assistant for the c / r keys?",
		"setup.ai.auto":          "auto (first installed)",
		"setup.ai.none":          "none found on PATH; auto stays empty until you install one.",
		"setup.hint.multi":       "↑↓ move · space toggle · enter next · esc cancel",
		"setup.hint.single":      "↑↓ move · enter next · esc cancel",
		"setup.hint.next":        "enter continue · esc cancel",
		"setup.hint.confirm":     "enter save & scan · esc cancel",
		"setup.confirm.title":    "Ready to write your config",
		"setup.row.roots":        "folders",
		"setup.row.editor":       "editor",
		"setup.row.ai":           "assistant",
		"setup.cancelled":        "hop: setup cancelled, nothing written",
		"setup.done":             "hop: ready, %d projects indexed",
		"setup.shell_hint":       "Last step, add this to %s then restart your shell:",
		"setup.shell.title":      "Shell integration",
		"setup.shell.prompt":     "Add the hop integration to %s? It loads on each new shell.",
		"setup.shell.already":    "Already present in %s, nothing to add.",
		"setup.shell.yes":        "Yes, add it for me",
		"setup.shell.no":         "No, I will add it myself",
		"setup.row.shell":        "shell",
		"setup.shellval.write":   "will add to %s",
		"setup.shellval.skip":    "skip (shown below)",
		"setup.shellval.already": "already configured",
		"setup.shell_done":       "Added the hop integration to %s. Restart your shell to use `p`.",
		"setup.shell_present":    "Shell integration already present in %s.",
		"setup.shell_failed":     "hop: could not write %s automatically.",
		"setup.shell_unknown": "hop does not know the shell in $SHELL (%s), so it cannot wire it for you.\n" +
			"The zsh line below is only a suggestion: adapt it to your shell's own startup file.",

		"cli.no_project":            "hop: no project for %q",
		"cli.no_index":              "hop: no project indexed, run `hop scan`",
		"cli.unsafe_path":           "hop: refusing a path with control characters",
		"cli.frequent_header":       "hop · %d projects (no interactive terminal, list fallback):",
		"cli.tip":                   "tip: p <keyword> [<keyword>...] to jump, p - to go back",
		"cli.no_prev":               "hop: no previous project",
		"cli.pruned":                "hop: removed %d dead path(s)",
		"cli.prune_failed":          "hop: could not remove dead paths: %v",
		"cli.pinned":                "hop: pinned %s",
		"cli.unpinned":              "hop: unpinned %s",
		"cli.tracked":               "hop: now tracking %s",
		"cli.untracked":             "hop: no longer tracking %s",
		"cli.track_already":         "hop: %s is already tracked",
		"cli.track_not_found":       "hop: %s was not tracked",
		"cli.track_not_dir":         "hop: %s is not a directory",
		"cli.import_no_zoxide":      "hop: zoxide not found on PATH; install it first (e.g. brew install zoxide)",
		"cli.import_failed":         "hop: zoxide import failed: %v",
		"cli.import_done":           "hop: imported %d, tracked %d, skipped %d",
		"cli.import_dry":            "hop: would import %d, track %d, skip %d (dry run, nothing written)",
		"cli.import_unknown_source": "hop: unknown import source %q (only zoxide is supported)",
		"cli.import_unknown_flag":   "hop: unknown flag %q for import",
		"cli.config_created":        "hop: config created at %s",
		"cli.config_saved":          "hop: config saved → %s",
		"cli.scan_summary":          "hop: %d projects indexed in %d categories",
		"cli.indexing":              "hop: initial indexing…",
		"cli.doctor.root":           "root",
		"cli.doctor.bin":            "binary",
		"cli.doctor.index_missing":  "MISSING (run `hop scan`)",
		"cli.doctor.title":          "hop doctor",
		"cli.doctor.ok":             "OK",
		"cli.doctor.none_found":     "(none found)",
		"cli.doctor.active":         "active: %s",
		"cli.doctor.none":           "none",
		"cli.doctor.editor":         "editor",
		"cli.doctor.ai":             "ai",
		"cli.doctor.action":         "action",
		"cli.doctor.index":          "index",
		"cli.init_cmd_no_name":      "hop: --cmd requires a name",
		"cli.init_unknown_flag":     "hop: unknown flag %q",
		"cli.init_invalid_name":     "hop: invalid function name %q (letters, digits, underscore; not starting with a digit)",
		"cli.init_invalid_config":   "hop: ignoring invalid [shell] command %q in %s, using %q instead",
		"cli.init_cmd_is_shell": "hop: --cmd %s is ambiguous: %s is a shell name, not a function name.\n" +
			"  did you mean  hop init %s  (default name %q), or  hop init %s --cmd NAME ?",
		"cli.init_unsupported_shell": "hop: unsupported shell %q (zsh, bash, fish)",
		"cli.import_from_no_source":  "hop: --from requires a source (only zoxide is supported)",
		"cli.pin_usage":              "hop: usage: hop pin <keyword>",
		"cli.unpin_usage":            "hop: usage: hop unpin <keyword>",
		"cli.nav_unknown_option":     "hop: unknown option %q",
		"cli.nav_jump_usage":         "usage: p - (previous), p -2, p -3 (nth most recent)",
		"cli.unknown_command":        "hop: unknown command %q (try `hop help`)",
		"cli.help": `hop · project switcher

Usage:
  hop setup              Guided first-run setup (folders, editor, assistant)
  hop nav [keyword...]   Resolve keywords and print the target (used by p)
  hop query [keyword...] Print the best match path (plain, for scripts; --list for all)
  hop scan               (Re)build the project index
  hop add <path>         Record a visit (frecency)
  hop init zsh [--cmd N] Print the shell integration (function "p" by default)
  hop config             Interactive configuration editor
  hop pin <keyword>      Pin the matching project to the top of the Hub
  hop unpin <keyword>    Remove a pin
  hop track <path>       Add a folder to the search list (even without git)
  hop untrack <path>     Remove a folder from the search list
  hop import --from zoxide  Seed ranking from zoxide (--dry-run to preview)
  hop clean              Forget projects whose folder no longer exists
  hop doctor             Configuration diagnostics
  hop version            Print the version

Daily, after  eval "$(hop init zsh)"  in ~/.zshrc:
  p <keyword>            jump straight to the best project
  p <keyword> <keyword>  narrow by sub-path (e.g. p acme web)
  p -                    go back to the previous project
  p                      open the interactive Hub (fuzzy list, Enter = cd)
`,
	},
	FR: {
		"hub.hint.tab":     "↑↓ naviguer · entrée cd · tab actions · échap annuler",
		"hub.hint.shift":   "↑↓ naviguer · entrée cd · tab menu · échap annuler",
		"hub.hint.enter":   "↑↓ naviguer · entrée actions · échap annuler",
		"hub.shift_prefix": "maj → ",
		"hub.no_match":     "(aucun match)",
		"hub.git.none":     "pas de dépôt git",
		"hub.git.clean":    "propre",
		"hub.git.dirty":    "modifié",
		"hub.actions.hint": "échap/tab retour · ctrl+c annuler",

		"time.now":     "à l'instant",
		"time.fmt":     "il y a %s",
		"time.u.min":   "min",
		"time.u.hour":  "h",
		"time.u.day":   "j",
		"time.u.week":  "sem",
		"time.u.month": "mois",
		"time.u.year":  "an",

		"action.cd":          "cd ici",
		"action.editor":      "ouvrir dans l'éditeur",
		"action.ai":          "lancer %s",
		"action.ai.resume":   "reprendre %s",
		"action.git":         "git status",
		"action.remote":      "ouvrir le repo distant",
		"action.finder":      "ouvrir dans le Finder",
		"action.filemanager": "ouvrir dans le gestionnaire de fichiers",
		"action.mux":         "session %s",

		"action.short.resume":      "reprise",
		"action.short.pin":         "épingler",
		"action.short.unpin":       "désépingler",
		"action.pin":               "ajouter aux favoris",
		"action.unpin":             "retirer des favoris",
		"action.short.remote":      "distant",
		"action.short.filemanager": "fichiers",

		"config.title":          "hop · configuration",
		"config.field.command":  "Nom de la commande",
		"config.field.theme":    "Thème",
		"config.field.access":   "Accès aux actions du Hub",
		"config.field.editor":   "Éditeur (action ouvrir)",
		"config.field.tmux":     "Afficher l'action tmux",
		"config.field.language": "Langue",
		"config.field.roots":    "Racines de scan",
		"config.field.depth":    "Profondeur de scan",
		"config.field.ignore":   "Dossiers ignorés",
		"config.help.command":   "le raccourci quotidien (p) ; effet au prochain exec zsh",
		"config.help.access":    "tab : Tab ouvre le menu · shift : MAJUSCULE direct · enter : Entrée ouvre le menu",
		"config.help.editor":    "commande lancée par l'action éditeur",
		"config.help.roots":     "dossiers séparés par des espaces",
		"config.help.depth":     "nombre de niveaux sous chaque racine",
		"config.help.ignore":    "séparés par des espaces",
		"config.opt.no":         "non",
		"config.opt.yes":        "oui",
		"config.hint":           "↑↓ champ · ←→ valeur · taper pour éditer · entrée/ctrl+s sauver · échap annuler",

		"cli.setup_hint":         "hop : aucun projet indexé pour l'instant, lance `hop setup` pour choisir tes dossiers de projets",
		"setup.title":            "hop · installation",
		"setup.roots.title":      "Quels dossiers contiennent tes projets ?",
		"setup.roots.empty":      "Aucun dossier de projets courant trouvé dans ton home. Tu pourras ajouter des racines plus tard avec hop config.",
		"setup.roots.repos":      "%d dépôts",
		"setup.editor.title":     "Ouvrir les projets dans quel éditeur ?",
		"setup.editor.empty":     "Aucun éditeur connu trouvé dans le PATH. Tu pourras en définir un plus tard avec hop config.",
		"setup.ai.title":         "Quel assistant IA pour les touches c / r ?",
		"setup.ai.auto":          "auto (le premier installé)",
		"setup.ai.none":          "aucun trouvé dans le PATH ; auto restera vide tant que tu n'en installes pas.",
		"setup.hint.multi":       "↑↓ naviguer · espace cocher · entrée suivant · échap annuler",
		"setup.hint.single":      "↑↓ naviguer · entrée suivant · échap annuler",
		"setup.hint.next":        "entrée continuer · échap annuler",
		"setup.hint.confirm":     "entrée sauver & scanner · échap annuler",
		"setup.confirm.title":    "Prêt à écrire ta configuration",
		"setup.row.roots":        "dossiers",
		"setup.row.editor":       "éditeur",
		"setup.row.ai":           "assistant",
		"setup.cancelled":        "hop : installation annulée, rien n'a été écrit",
		"setup.done":             "hop : prêt, %d projets indexés",
		"setup.shell_hint":       "Dernière étape, ajoute ceci à %s puis relance ton shell :",
		"setup.shell.title":      "Intégration shell",
		"setup.shell.prompt":     "Ajouter l'intégration hop à %s ? Elle se charge à chaque nouveau shell.",
		"setup.shell.already":    "Déjà présente dans %s, rien à ajouter.",
		"setup.shell.yes":        "Oui, ajoute-la pour moi",
		"setup.shell.no":         "Non, je l'ajoute moi-même",
		"setup.row.shell":        "shell",
		"setup.shellval.write":   "ajout dans %s",
		"setup.shellval.skip":    "ignorer (affiché plus bas)",
		"setup.shellval.already": "déjà configuré",
		"setup.shell_done":       "Intégration hop ajoutée à %s. Relance ton shell pour utiliser `p`.",
		"setup.shell_present":    "Intégration shell déjà présente dans %s.",
		"setup.shell_failed":     "hop : impossible d'écrire %s automatiquement.",
		"setup.shell_unknown": "hop ne connaît pas le shell indiqué dans $SHELL (%s), il ne peut donc pas le configurer pour toi.\n" +
			"La ligne zsh ci-dessous n'est qu'une suggestion : adapte-la au fichier de démarrage de ton propre shell.",

		"cli.no_project":            "hop: aucun projet pour %q",
		"cli.no_index":              "hop: aucun projet indexé, lance `hop scan`",
		"cli.unsafe_path":           "hop : chemin contenant des caractères de contrôle, action refusée",
		"cli.frequent_header":       "hop · %d projets (pas de terminal interactif, repli liste) :",
		"cli.tip":                   "astuce : p <mot-clé> [<mot-clé>...] pour sauter, p - pour revenir",
		"cli.no_prev":               "hop: pas de projet précédent",
		"cli.pruned":                "hop: %d chemin(s) mort(s) supprimé(s)",
		"cli.prune_failed":          "hop: impossible de supprimer les chemins morts : %v",
		"cli.pinned":                "hop: %s épinglé",
		"cli.unpinned":              "hop: %s désépinglé",
		"cli.tracked":               "hop: %s ajouté à la liste de recherche",
		"cli.untracked":             "hop: %s retiré de la liste de recherche",
		"cli.track_already":         "hop: %s déjà suivi",
		"cli.track_not_found":       "hop: %s n'était pas suivi",
		"cli.track_not_dir":         "hop: %s n'est pas un dossier",
		"cli.import_no_zoxide":      "hop: zoxide introuvable dans le PATH ; installe-le d'abord (ex. brew install zoxide)",
		"cli.import_failed":         "hop: échec de l'import zoxide : %v",
		"cli.import_done":           "hop: %d importés, %d suivis, %d ignorés",
		"cli.import_dry":            "hop: %d à importer, %d à suivre, %d à ignorer (simulation, rien écrit)",
		"cli.import_unknown_source": "hop: source d'import inconnue %q (seul zoxide est supporté)",
		"cli.import_unknown_flag":   "hop: option inconnue %q pour import",
		"cli.config_created":        "hop: config créée dans %s",
		"cli.config_saved":          "hop: config sauvegardée → %s",
		"cli.scan_summary":          "hop: %d projets indexés dans %d catégories",
		"cli.indexing":              "hop: indexation initiale…",
		"cli.doctor.root":           "racine",
		"cli.doctor.bin":            "binaire",
		"cli.doctor.index_missing":  "ABSENT (lance `hop scan`)",
		"cli.doctor.title":          "hop doctor",
		"cli.doctor.ok":             "OK",
		"cli.doctor.none_found":     "(aucun trouvé)",
		"cli.doctor.active":         "actif : %s",
		"cli.doctor.none":           "aucun",
		"cli.doctor.editor":         "éditeur",
		"cli.doctor.ai":             "ia",
		"cli.doctor.action":         "action",
		"cli.doctor.index":          "index",
		"cli.init_cmd_no_name":      "hop : --cmd attend un nom",
		"cli.init_unknown_flag":     "hop : option inconnue %q",
		"cli.init_invalid_name":     "hop : nom de fonction invalide %q (lettres, chiffres, souligné ; ne commence pas par un chiffre)",
		"cli.init_invalid_config":   "hop : commande [shell] invalide %q dans %s, ignorée, %q utilisé à la place",
		"cli.init_cmd_is_shell": "hop : --cmd %s est ambigu : %s est un nom de shell, pas un nom de fonction.\n" +
			"  voulais-tu  hop init %s  (nom par défaut %q), ou  hop init %s --cmd NOM ?",
		"cli.init_unsupported_shell": "hop : shell non supporté %q (zsh, bash, fish)",
		"cli.import_from_no_source":  "hop : --from attend une source (seul zoxide est supporté)",
		"cli.pin_usage":              "hop : utilisation : hop pin <mot-clé>",
		"cli.unpin_usage":            "hop : utilisation : hop unpin <mot-clé>",
		"cli.nav_unknown_option":     "hop : option inconnue %q",
		"cli.nav_jump_usage":         "utilisation : p - (précédent), p -2, p -3 (n-ième plus récent)",
		"cli.unknown_command":        "hop: commande inconnue %q (essaie `hop help`)",
		"cli.help": `hop · commutateur de projets

Usage:
  hop setup              Configuration guidée au premier lancement (dossiers, éditeur, assistant)
  hop nav [mot-clé...]   Résout des mots-clés et imprime la cible (utilisé par p)
  hop query [mot-clé...] Imprime le chemin du meilleur match (brut, pour scripts ; --list pour tous)
  hop scan               (Re)construit l'index des projets
  hop add <path>         Enregistre un accès (frécence)
  hop init zsh [--cmd N] Imprime l'intégration shell (fonction "p" par défaut)
  hop config             Éditeur de configuration interactif
  hop pin <mot-clé>      Épingle le projet correspondant en tête du Hub
  hop unpin <mot-clé>    Retire un épinglage
  hop track <chemin>     Ajoute un dossier à la liste de recherche (même sans git)
  hop untrack <chemin>   Retire un dossier de la liste de recherche
  hop import --from zoxide  Amorce le classement depuis zoxide (--dry-run pour prévisualiser)
  hop clean              Oublie les projets dont le dossier n'existe plus
  hop doctor             Diagnostic de configuration
  hop version            Affiche la version

Au quotidien, après  eval "$(hop init zsh)"  dans ~/.zshrc :
  p <mot-clé>            saut direct vers le meilleur projet
  p <mot-clé> <mot-clé>  affine par sous-chemin (ex. p acme web)
  p -                    revient au projet précédent
  p                      ouvre le Hub interactif (liste fuzzy, Entrée = cd)
`,
	},
	ES: {
		"hub.hint.tab":     "↑↓ navegar · enter cd · tab acciones · esc cancelar",
		"hub.hint.shift":   "↑↓ navegar · enter cd · tab menú · esc cancelar",
		"hub.hint.enter":   "↑↓ navegar · enter acciones · esc cancelar",
		"hub.shift_prefix": "mayús → ",
		"hub.no_match":     "(sin coincidencias)",
		"hub.git.none":     "sin repo git",
		"hub.git.clean":    "limpio",
		"hub.git.dirty":    "modificado",
		"hub.actions.hint": "esc/tab volver · ctrl+c cancelar",

		"time.now":     "ahora",
		"time.fmt":     "hace %s",
		"time.u.min":   "min",
		"time.u.hour":  "h",
		"time.u.day":   "d",
		"time.u.week":  "sem",
		"time.u.month": "mes",
		"time.u.year":  "a",

		"action.cd":          "cd aquí",
		"action.editor":      "abrir en el editor",
		"action.ai":          "lanzar %s",
		"action.ai.resume":   "reanudar %s",
		"action.git":         "git status",
		"action.remote":      "abrir repo remoto",
		"action.finder":      "abrir en Finder",
		"action.filemanager": "abrir en el gestor de archivos",
		"action.mux":         "sesión %s",

		"action.short.resume":      "reanudar",
		"action.short.pin":         "fijar",
		"action.short.unpin":       "desfijar",
		"action.pin":               "añadir a favoritos",
		"action.unpin":             "quitar de favoritos",
		"action.short.remote":      "remoto",
		"action.short.filemanager": "archivos",

		"config.title":          "hop · configuración",
		"config.field.command":  "Nombre del comando",
		"config.field.theme":    "Tema",
		"config.field.access":   "Acceso a las acciones del Hub",
		"config.field.editor":   "Editor (acción abrir)",
		"config.field.tmux":     "Mostrar la acción tmux",
		"config.field.language": "Idioma",
		"config.field.roots":    "Raíces de escaneo",
		"config.field.depth":    "Profundidad de escaneo",
		"config.field.ignore":   "Carpetas ignoradas",
		"config.help.command":   "el atajo diario (p); efecto en el próximo exec zsh",
		"config.help.access":    "tab: Tab abre el menú · shift: MAYÚSCULA directa · enter: Enter abre el menú",
		"config.help.editor":    "comando que lanza la acción de editor",
		"config.help.roots":     "carpetas separadas por espacios",
		"config.help.depth":     "niveles bajo cada raíz",
		"config.help.ignore":    "separadas por espacios",
		"config.opt.no":         "no",
		"config.opt.yes":        "sí",
		"config.hint":           "↑↓ campo · ←→ valor · escribe para editar · enter/ctrl+s guardar · esc cancelar",

		"cli.setup_hint":         "hop: aún no hay proyectos indexados, ejecuta `hop setup` para elegir tus carpetas de proyectos",
		"setup.title":            "hop · instalación",
		"setup.roots.title":      "¿Qué carpetas contienen tus proyectos?",
		"setup.roots.empty":      "No se encontró ninguna carpeta de proyectos común en tu home. Puedes añadir raíces luego con hop config.",
		"setup.roots.repos":      "%d repos",
		"setup.editor.title":     "¿En qué editor abrir los proyectos?",
		"setup.editor.empty":     "No se encontró ningún editor conocido en el PATH. Puedes definir uno luego con hop config.",
		"setup.ai.title":         "¿Qué asistente de IA para las teclas c / r?",
		"setup.ai.auto":          "auto (el primero instalado)",
		"setup.ai.none":          "ninguno encontrado en el PATH; auto queda vacío hasta que instales uno.",
		"setup.hint.multi":       "↑↓ mover · espacio marcar · enter siguiente · esc cancelar",
		"setup.hint.single":      "↑↓ mover · enter siguiente · esc cancelar",
		"setup.hint.next":        "enter continuar · esc cancelar",
		"setup.hint.confirm":     "enter guardar y escanear · esc cancelar",
		"setup.confirm.title":    "Listo para escribir tu configuración",
		"setup.row.roots":        "carpetas",
		"setup.row.editor":       "editor",
		"setup.row.ai":           "asistente",
		"setup.cancelled":        "hop: instalación cancelada, no se escribió nada",
		"setup.done":             "hop: listo, %d proyectos indexados",
		"setup.shell_hint":       "Último paso, añade esto a %s y reinicia tu shell:",
		"setup.shell.title":      "Integración del shell",
		"setup.shell.prompt":     "¿Añadir la integración de hop a %s? Se carga en cada nuevo shell.",
		"setup.shell.already":    "Ya presente en %s, nada que añadir.",
		"setup.shell.yes":        "Sí, añádela por mí",
		"setup.shell.no":         "No, la añado yo mismo",
		"setup.row.shell":        "shell",
		"setup.shellval.write":   "se añadirá a %s",
		"setup.shellval.skip":    "omitir (se muestra abajo)",
		"setup.shellval.already": "ya configurado",
		"setup.shell_done":       "Integración de hop añadida a %s. Reinicia tu shell para usar `p`.",
		"setup.shell_present":    "Integración del shell ya presente en %s.",
		"setup.shell_failed":     "hop: no se pudo escribir %s automáticamente.",
		"setup.shell_unknown": "hop no conoce el shell indicado en $SHELL (%s), así que no puede configurarlo por ti.\n" +
			"La línea zsh de abajo es solo una sugerencia: adáptala al archivo de inicio de tu propio shell.",

		"cli.no_project":            "hop: ningún proyecto para %q",
		"cli.no_index":              "hop: ningún proyecto indexado, ejecuta `hop scan`",
		"cli.unsafe_path":           "hop: ruta con caracteres de control, acción rechazada",
		"cli.frequent_header":       "hop · %d proyectos (sin terminal interactiva, lista de respaldo):",
		"cli.tip":                   "consejo: p <palabra> [<palabra>...] para saltar, p - para volver",
		"cli.no_prev":               "hop: sin proyecto anterior",
		"cli.pruned":                "hop: %d ruta(s) muerta(s) eliminada(s)",
		"cli.prune_failed":          "hop: no se pudieron eliminar las rutas muertas: %v",
		"cli.pinned":                "hop: %s fijado",
		"cli.unpinned":              "hop: %s desfijado",
		"cli.tracked":               "hop: %s añadido a la lista de búsqueda",
		"cli.untracked":             "hop: %s eliminado de la lista de búsqueda",
		"cli.track_already":         "hop: %s ya está en seguimiento",
		"cli.track_not_found":       "hop: %s no estaba en seguimiento",
		"cli.track_not_dir":         "hop: %s no es un directorio",
		"cli.import_no_zoxide":      "hop: zoxide no está en el PATH; instálalo primero (p. ej. brew install zoxide)",
		"cli.import_failed":         "hop: falló la importación de zoxide: %v",
		"cli.import_done":           "hop: %d importados, %d en seguimiento, %d omitidos",
		"cli.import_dry":            "hop: %d a importar, %d a seguir, %d a omitir (simulación, no se escribió nada)",
		"cli.import_unknown_source": "hop: fuente de importación desconocida %q (solo se admite zoxide)",
		"cli.import_unknown_flag":   "hop: opción desconocida %q para import",
		"cli.config_created":        "hop: config creada en %s",
		"cli.config_saved":          "hop: config guardada → %s",
		"cli.scan_summary":          "hop: %d proyectos indexados en %d categorías",
		"cli.indexing":              "hop: indexación inicial…",
		"cli.doctor.root":           "raíz",
		"cli.doctor.bin":            "binario",
		"cli.doctor.index_missing":  "AUSENTE (ejecuta `hop scan`)",
		"cli.doctor.title":          "hop doctor",
		"cli.doctor.ok":             "OK",
		"cli.doctor.none_found":     "(ninguno encontrado)",
		"cli.doctor.active":         "activo: %s",
		"cli.doctor.none":           "ninguno",
		"cli.doctor.editor":         "editor",
		"cli.doctor.ai":             "ia",
		"cli.doctor.action":         "acción",
		"cli.doctor.index":          "índice",
		"cli.init_cmd_no_name":      "hop: --cmd requiere un nombre",
		"cli.init_unknown_flag":     "hop: opción desconocida %q",
		"cli.init_invalid_name":     "hop: nombre de función inválido %q (letras, dígitos, guion bajo; no empieza por un dígito)",
		"cli.init_invalid_config":   "hop: se ignora el command [shell] inválido %q en %s, se usa %q en su lugar",
		"cli.init_cmd_is_shell": "hop: --cmd %s es ambiguo: %s es un nombre de shell, no un nombre de función.\n" +
			"  ¿querías  hop init %s  (nombre por defecto %q), o  hop init %s --cmd NOMBRE ?",
		"cli.init_unsupported_shell": "hop: shell no soportado %q (zsh, bash, fish)",
		"cli.import_from_no_source":  "hop: --from requiere una fuente (solo se admite zoxide)",
		"cli.pin_usage":              "hop: uso: hop pin <palabra>",
		"cli.unpin_usage":            "hop: uso: hop unpin <palabra>",
		"cli.nav_unknown_option":     "hop: opción desconocida %q",
		"cli.nav_jump_usage":         "uso: p - (anterior), p -2, p -3 (n-ésimo más reciente)",
		"cli.unknown_command":        "hop: comando desconocido %q (prueba `hop help`)",
		"cli.help": `hop · conmutador de proyectos

Uso:
  hop setup              Configuración guiada inicial (carpetas, editor, asistente)
  hop nav [palabra...]   Resuelve palabras e imprime el destino (usado por p)
  hop query [palabra...] Imprime la ruta del mejor resultado (texto plano, para scripts; --list para todos)
  hop scan               (Re)construye el índice de proyectos
  hop add <path>         Registra un acceso (frecencia)
  hop init zsh [--cmd N] Imprime la integración del shell (función "p" por defecto)
  hop config             Editor de configuración interactivo
  hop pin <palabra>      Fija el proyecto correspondiente arriba del Hub
  hop unpin <palabra>    Quita una fijación
  hop track <ruta>       Añade una carpeta a la lista de búsqueda (aunque sin git)
  hop untrack <ruta>     Quita una carpeta de la lista de búsqueda
  hop import --from zoxide  Inicializa el ranking desde zoxide (--dry-run para previsualizar)
  hop clean              Olvida proyectos cuya carpeta ya no existe
  hop doctor             Diagnóstico de configuración
  hop version            Muestra la versión

A diario, tras  eval "$(hop init zsh)"  en ~/.zshrc:
  p <palabra>            salta directo al mejor proyecto
  p <palabra> <palabra>  afina por sub-ruta (ej. p acme web)
  p -                    vuelve al proyecto anterior
  p                      abre el Hub interactivo (lista fuzzy, Enter = cd)
`,
	},
	PT: {
		"hub.hint.tab":     "↑↓ navegar · enter cd · tab ações · esc cancelar",
		"hub.hint.shift":   "↑↓ navegar · enter cd · tab menu · esc cancelar",
		"hub.hint.enter":   "↑↓ navegar · enter ações · esc cancelar",
		"hub.shift_prefix": "shift → ",
		"hub.no_match":     "(sem correspondência)",
		"hub.git.none":     "sem repo git",
		"hub.git.clean":    "limpo",
		"hub.git.dirty":    "modificado",
		"hub.actions.hint": "esc/tab voltar · ctrl+c cancelar",

		"time.now":     "agora",
		"time.fmt":     "há %s",
		"time.u.min":   "min",
		"time.u.hour":  "h",
		"time.u.day":   "d",
		"time.u.week":  "sem",
		"time.u.month": "mês",
		"time.u.year":  "a",

		"action.cd":          "cd aqui",
		"action.editor":      "abrir no editor",
		"action.ai":          "abrir %s",
		"action.ai.resume":   "retomar %s",
		"action.git":         "git status",
		"action.remote":      "abrir repo remoto",
		"action.finder":      "abrir no Finder",
		"action.filemanager": "abrir no gerenciador de arquivos",
		"action.mux":         "sessão %s",

		"action.short.resume":      "retomar",
		"action.short.pin":         "fixar",
		"action.short.unpin":       "desafixar",
		"action.pin":               "adicionar aos favoritos",
		"action.unpin":             "remover dos favoritos",
		"action.short.remote":      "remoto",
		"action.short.filemanager": "arquivos",

		"config.title":          "hop · configuração",
		"config.field.command":  "Nome do comando",
		"config.field.theme":    "Tema",
		"config.field.access":   "Acesso às ações do Hub",
		"config.field.editor":   "Editor (ação abrir)",
		"config.field.tmux":     "Mostrar a ação tmux",
		"config.field.language": "Idioma",
		"config.field.roots":    "Raízes de varredura",
		"config.field.depth":    "Profundidade de varredura",
		"config.field.ignore":   "Pastas ignoradas",
		"config.help.command":   "o atalho diário (p); efeito no próximo exec zsh",
		"config.help.access":    "tab: Tab abre o menu · shift: MAIÚSCULA direta · enter: Enter abre o menu",
		"config.help.editor":    "comando executado pela ação de editor",
		"config.help.roots":     "pastas separadas por espaços",
		"config.help.depth":     "níveis abaixo de cada raiz",
		"config.help.ignore":    "separadas por espaços",
		"config.opt.no":         "não",
		"config.opt.yes":        "sim",
		"config.hint":           "↑↓ campo · ←→ valor · digite para editar · enter/ctrl+s salvar · esc cancelar",

		"cli.setup_hint":         "hop: ainda não há projetos indexados, execute `hop setup` para escolher suas pastas de projetos",
		"setup.title":            "hop · instalação",
		"setup.roots.title":      "Quais pastas contêm seus projetos?",
		"setup.roots.empty":      "Nenhuma pasta de projetos comum encontrada no seu home. Você pode adicionar raízes depois com hop config.",
		"setup.roots.repos":      "%d repos",
		"setup.editor.title":     "Abrir os projetos em qual editor?",
		"setup.editor.empty":     "Nenhum editor conhecido encontrado no PATH. Você pode definir um depois com hop config.",
		"setup.ai.title":         "Qual assistente de IA para as teclas c / r?",
		"setup.ai.auto":          "auto (o primeiro instalado)",
		"setup.ai.none":          "nenhum encontrado no PATH; auto fica vazio até você instalar um.",
		"setup.hint.multi":       "↑↓ mover · espaço marcar · enter próximo · esc cancelar",
		"setup.hint.single":      "↑↓ mover · enter próximo · esc cancelar",
		"setup.hint.next":        "enter continuar · esc cancelar",
		"setup.hint.confirm":     "enter salvar e escanear · esc cancelar",
		"setup.confirm.title":    "Pronto para escrever sua configuração",
		"setup.row.roots":        "pastas",
		"setup.row.editor":       "editor",
		"setup.row.ai":           "assistente",
		"setup.cancelled":        "hop: instalação cancelada, nada foi escrito",
		"setup.done":             "hop: pronto, %d projetos indexados",
		"setup.shell_hint":       "Último passo, adicione isto a %s e reinicie seu shell:",
		"setup.shell.title":      "Integração do shell",
		"setup.shell.prompt":     "Adicionar a integração do hop a %s? Carrega a cada novo shell.",
		"setup.shell.already":    "Já presente em %s, nada a adicionar.",
		"setup.shell.yes":        "Sim, adicione por mim",
		"setup.shell.no":         "Não, eu adiciono",
		"setup.row.shell":        "shell",
		"setup.shellval.write":   "será adicionado a %s",
		"setup.shellval.skip":    "pular (mostrado abaixo)",
		"setup.shellval.already": "já configurado",
		"setup.shell_done":       "Integração do hop adicionada a %s. Reinicie seu shell para usar `p`.",
		"setup.shell_present":    "Integração do shell já presente em %s.",
		"setup.shell_failed":     "hop: não foi possível escrever %s automaticamente.",
		"setup.shell_unknown": "hop não conhece o shell indicado em $SHELL (%s), então não pode configurá-lo para você.\n" +
			"A linha zsh abaixo é apenas uma sugestão: adapte-a ao arquivo de inicialização do seu próprio shell.",

		"cli.no_project":            "hop: nenhum projeto para %q",
		"cli.no_index":              "hop: nenhum projeto indexado, execute `hop scan`",
		"cli.unsafe_path":           "hop: caminho com caracteres de controle, ação recusada",
		"cli.frequent_header":       "hop · %d projetos (sem terminal interativo, lista alternativa):",
		"cli.tip":                   "dica: p <palavra> [<palavra>...] para saltar, p - para voltar",
		"cli.no_prev":               "hop: sem projeto anterior",
		"cli.pruned":                "hop: %d caminho(s) morto(s) removido(s)",
		"cli.prune_failed":          "hop: não foi possível remover os caminhos mortos: %v",
		"cli.pinned":                "hop: %s fixado",
		"cli.unpinned":              "hop: %s desafixado",
		"cli.tracked":               "hop: %s adicionado à lista de busca",
		"cli.untracked":             "hop: %s removido da lista de busca",
		"cli.track_already":         "hop: %s já está na lista",
		"cli.track_not_found":       "hop: %s não estava na lista",
		"cli.track_not_dir":         "hop: %s não é um diretório",
		"cli.import_no_zoxide":      "hop: zoxide não encontrado no PATH; instale-o primeiro (ex. brew install zoxide)",
		"cli.import_failed":         "hop: falha na importação do zoxide: %v",
		"cli.import_done":           "hop: %d importados, %d na lista, %d ignorados",
		"cli.import_dry":            "hop: %d a importar, %d a rastrear, %d a ignorar (simulação, nada gravado)",
		"cli.import_unknown_source": "hop: fonte de importação desconhecida %q (apenas zoxide é suportado)",
		"cli.import_unknown_flag":   "hop: opção desconhecida %q para import",
		"cli.config_created":        "hop: config criada em %s",
		"cli.config_saved":          "hop: config salva → %s",
		"cli.scan_summary":          "hop: %d projetos indexados em %d categorias",
		"cli.indexing":              "hop: indexação inicial…",
		"cli.doctor.root":           "raiz",
		"cli.doctor.bin":            "binário",
		"cli.doctor.index_missing":  "AUSENTE (execute `hop scan`)",
		"cli.doctor.title":          "hop doctor",
		"cli.doctor.ok":             "OK",
		"cli.doctor.none_found":     "(nenhum encontrado)",
		"cli.doctor.active":         "ativo: %s",
		"cli.doctor.none":           "nenhum",
		"cli.doctor.editor":         "editor",
		"cli.doctor.ai":             "ia",
		"cli.doctor.action":         "ação",
		"cli.doctor.index":          "índice",
		"cli.init_cmd_no_name":      "hop: --cmd exige um nome",
		"cli.init_unknown_flag":     "hop: opção desconhecida %q",
		"cli.init_invalid_name":     "hop: nome de função inválido %q (letras, dígitos, sublinhado; não começa por um dígito)",
		"cli.init_invalid_config":   "hop: command [shell] inválido %q em %s ignorado, %q usado em seu lugar",
		"cli.init_cmd_is_shell": "hop: --cmd %s é ambíguo: %s é um nome de shell, não um nome de função.\n" +
			"  você quis dizer  hop init %s  (nome padrão %q), ou  hop init %s --cmd NOME ?",
		"cli.init_unsupported_shell": "hop: shell não suportado %q (zsh, bash, fish)",
		"cli.import_from_no_source":  "hop: --from exige uma fonte (apenas zoxide é suportado)",
		"cli.pin_usage":              "hop: uso: hop pin <palavra>",
		"cli.unpin_usage":            "hop: uso: hop unpin <palavra>",
		"cli.nav_unknown_option":     "hop: opção desconhecida %q",
		"cli.nav_jump_usage":         "uso: p - (anterior), p -2, p -3 (n-ésimo mais recente)",
		"cli.unknown_command":        "hop: comando desconhecido %q (tente `hop help`)",
		"cli.help": `hop · alternador de projetos

Uso:
  hop setup              Configuração guiada inicial (pastas, editor, assistente)
  hop nav [palavra...]   Resolve palavras e imprime o destino (usado por p)
  hop query [palavra...] Imprime o caminho do melhor resultado (texto puro, para scripts; --list para todos)
  hop scan               (Re)constrói o índice de projetos
  hop add <path>         Registra um acesso (frecência)
  hop init zsh [--cmd N] Imprime a integração do shell (função "p" por padrão)
  hop config             Editor de configuração interativo
  hop pin <palavra>      Fixa o projeto correspondente no topo do Hub
  hop unpin <palavra>    Remove uma fixação
  hop track <caminho>    Adiciona uma pasta à lista de busca (mesmo sem git)
  hop untrack <caminho>  Remove uma pasta da lista de busca
  hop import --from zoxide  Inicializa o ranking a partir do zoxide (--dry-run para pré-visualizar)
  hop clean              Esquece projetos cuja pasta não existe mais
  hop doctor             Diagnóstico de configuração
  hop version            Mostra a versão

No dia a dia, após  eval "$(hop init zsh)"  em ~/.zshrc:
  p <palavra>            salta direto para o melhor projeto
  p <palavra> <palavra>  refina por sub-caminho (ex. p acme web)
  p -                    volta ao projeto anterior
  p                      abre o Hub interativo (lista fuzzy, Enter = cd)
`,
	},
}

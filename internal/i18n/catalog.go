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

		"action.cd":        "cd here",
		"action.editor":    "open in editor",
		"action.ai":        "launch %s",
		"action.ai.resume": "resume %s",
		"action.git":       "git status",
		"action.remote":    "open remote repo",
		"action.finder":    "open in Finder",
		"action.tmux":      "tmux session",

		"action.short.resume": "resume",
		"action.short.remote": "remote",

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

		"cli.no_project":           "hop: no project for %q",
		"cli.no_index":             "hop: no project indexed, run `hop scan`",
		"cli.frequent_header":      "hop · %d projects (no interactive terminal, list fallback):",
		"cli.tip":                  "tip: p <keyword> [<keyword>...] to jump, p - to go back",
		"cli.no_prev":              "hop: no previous project",
		"cli.pruned":               "hop: removed %d dead path(s)",
		"cli.pinned":               "hop: pinned %s",
		"cli.unpinned":             "hop: unpinned %s",
		"cli.config_created":       "hop: config created at %s",
		"cli.config_saved":         "hop: config saved → %s",
		"cli.scan_summary":         "hop: %d projects indexed in %d categories",
		"cli.indexing":             "hop: initial indexing…",
		"cli.doctor.root":          "root",
		"cli.doctor.bin":           "binary",
		"cli.doctor.index_missing": "MISSING (run `hop scan`)",
		"cli.help": `hop · project switcher

Usage:
  hop nav [keyword...]   Resolve keywords and print the target (used by p)
  hop scan               (Re)build the project index
  hop add <path>         Record a visit (frecency)
  hop init zsh [--cmd N] Print the shell integration (function "p" by default)
  hop config             Interactive configuration editor
  hop pin <keyword>      Pin the matching project to the top of the Hub
  hop unpin <keyword>    Remove a pin
  hop clean              Forget projects whose folder no longer exists
  hop doctor             Configuration diagnostics
  hop version            Print the version

Daily, after  eval "$(hop init zsh)"  in ~/.zsh_init:
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

		"action.cd":        "cd ici",
		"action.editor":    "ouvrir dans l'éditeur",
		"action.ai":        "lancer %s",
		"action.ai.resume": "reprendre %s",
		"action.git":       "git status",
		"action.remote":    "ouvrir le repo distant",
		"action.finder":    "ouvrir dans le Finder",
		"action.tmux":      "session tmux",

		"action.short.resume": "reprise",
		"action.short.remote": "distant",

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

		"cli.no_project":           "hop: aucun projet pour %q",
		"cli.no_index":             "hop: aucun projet indexé, lance `hop scan`",
		"cli.frequent_header":      "hop · %d projets (pas de terminal interactif, repli liste) :",
		"cli.tip":                  "astuce : p <mot-clé> [<mot-clé>...] pour sauter, p - pour revenir",
		"cli.no_prev":              "hop: pas de projet précédent",
		"cli.pruned":               "hop: %d chemin(s) mort(s) supprimé(s)",
		"cli.pinned":               "hop: %s épinglé",
		"cli.unpinned":             "hop: %s désépinglé",
		"cli.config_created":       "hop: config créée dans %s",
		"cli.config_saved":         "hop: config sauvegardée → %s",
		"cli.scan_summary":         "hop: %d projets indexés dans %d catégories",
		"cli.indexing":             "hop: indexation initiale…",
		"cli.doctor.root":          "racine",
		"cli.doctor.bin":           "binaire",
		"cli.doctor.index_missing": "ABSENT (lance `hop scan`)",
		"cli.help": `hop · commutateur de projets

Usage:
  hop nav [mot-clé...]   Résout des mots-clés et imprime la cible (utilisé par p)
  hop scan               (Re)construit l'index des projets
  hop add <path>         Enregistre un accès (frécence)
  hop init zsh [--cmd N] Imprime l'intégration shell (fonction "p" par défaut)
  hop config             Éditeur de configuration interactif
  hop pin <mot-clé>      Épingle le projet correspondant en tête du Hub
  hop unpin <mot-clé>    Retire un épinglage
  hop clean              Oublie les projets dont le dossier n'existe plus
  hop doctor             Diagnostic de configuration
  hop version            Affiche la version

Au quotidien, après  eval "$(hop init zsh)"  dans ~/.zsh_init :
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

		"action.cd":        "cd aquí",
		"action.editor":    "abrir en el editor",
		"action.ai":        "lanzar %s",
		"action.ai.resume": "reanudar %s",
		"action.git":       "git status",
		"action.remote":    "abrir repo remoto",
		"action.finder":    "abrir en Finder",
		"action.tmux":      "sesión tmux",

		"action.short.resume": "reanudar",
		"action.short.remote": "remoto",

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

		"cli.no_project":           "hop: ningún proyecto para %q",
		"cli.no_index":             "hop: ningún proyecto indexado, ejecuta `hop scan`",
		"cli.frequent_header":      "hop · %d proyectos (sin terminal interactiva, lista de respaldo):",
		"cli.tip":                  "consejo: p <palabra> [<palabra>...] para saltar, p - para volver",
		"cli.no_prev":              "hop: sin proyecto anterior",
		"cli.pruned":               "hop: %d ruta(s) muerta(s) eliminada(s)",
		"cli.pinned":               "hop: %s fijado",
		"cli.unpinned":             "hop: %s desfijado",
		"cli.config_created":       "hop: config creada en %s",
		"cli.config_saved":         "hop: config guardada → %s",
		"cli.scan_summary":         "hop: %d proyectos indexados en %d categorías",
		"cli.indexing":             "hop: indexación inicial…",
		"cli.doctor.root":          "raíz",
		"cli.doctor.bin":           "binario",
		"cli.doctor.index_missing": "AUSENTE (ejecuta `hop scan`)",
		"cli.help": `hop · conmutador de proyectos

Uso:
  hop nav [palabra...]   Resuelve palabras e imprime el destino (usado por p)
  hop scan               (Re)construye el índice de proyectos
  hop add <path>         Registra un acceso (frecencia)
  hop init zsh [--cmd N] Imprime la integración del shell (función "p" por defecto)
  hop config             Editor de configuración interactivo
  hop pin <palabra>      Fija el proyecto correspondiente arriba del Hub
  hop unpin <palabra>    Quita una fijación
  hop clean              Olvida proyectos cuya carpeta ya no existe
  hop doctor             Diagnóstico de configuración
  hop version            Muestra la versión

A diario, tras  eval "$(hop init zsh)"  en ~/.zsh_init:
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

		"action.cd":        "cd aqui",
		"action.editor":    "abrir no editor",
		"action.ai":        "abrir %s",
		"action.ai.resume": "retomar %s",
		"action.git":       "git status",
		"action.remote":    "abrir repo remoto",
		"action.finder":    "abrir no Finder",
		"action.tmux":      "sessão tmux",

		"action.short.resume": "retomar",
		"action.short.remote": "remoto",

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

		"cli.no_project":           "hop: nenhum projeto para %q",
		"cli.no_index":             "hop: nenhum projeto indexado, execute `hop scan`",
		"cli.frequent_header":      "hop · %d projetos (sem terminal interativo, lista alternativa):",
		"cli.tip":                  "dica: p <palavra> [<palavra>...] para saltar, p - para voltar",
		"cli.no_prev":              "hop: sem projeto anterior",
		"cli.pruned":               "hop: %d caminho(s) morto(s) removido(s)",
		"cli.pinned":               "hop: %s fixado",
		"cli.unpinned":             "hop: %s desafixado",
		"cli.config_created":       "hop: config criada em %s",
		"cli.config_saved":         "hop: config salva → %s",
		"cli.scan_summary":         "hop: %d projetos indexados em %d categorias",
		"cli.indexing":             "hop: indexação inicial…",
		"cli.doctor.root":          "raiz",
		"cli.doctor.bin":           "binário",
		"cli.doctor.index_missing": "AUSENTE (execute `hop scan`)",
		"cli.help": `hop · alternador de projetos

Uso:
  hop nav [palavra...]   Resolve palavras e imprime o destino (usado por p)
  hop scan               (Re)constrói o índice de projetos
  hop add <path>         Registra um acesso (frecência)
  hop init zsh [--cmd N] Imprime a integração do shell (função "p" por padrão)
  hop config             Editor de configuração interativo
  hop pin <palavra>      Fixa o projeto correspondente no topo do Hub
  hop unpin <palavra>    Remove uma fixação
  hop clean              Esquece projetos cuja pasta não existe mais
  hop doctor             Diagnóstico de configuração
  hop version            Mostra a versão

No dia a dia, após  eval "$(hop init zsh)"  em ~/.zsh_init:
  p <palavra>            salta direto para o melhor projeto
  p <palavra> <palavra>  refina por sub-caminho (ex. p acme web)
  p -                    volta ao projeto anterior
  p                      abre o Hub interativo (lista fuzzy, Enter = cd)
`,
	},
}

package cli

// Core catalogs are fixed text, never AI-generated at runtime. Missing entries
// deliberately fall back to English; stable codes and JSON enums are untouched.
func secondaryLanguageIndex(language string) int {
	switch language {
	case "de":
		return 0
	case "fr":
		return 1
	case "es":
		return 2
	case "pt-BR":
		return 3
	case "ja":
		return 4
	}
	return -1
}

var coreTranslations = map[string][5]string{
	"Replace only a proven HAA-managed asset after fresh inspection; requires --target": {"Nur ein nachweislich von HAA verwaltetes Artefakt nach neuer Prüfung ersetzen; --target erforderlich", "Remplacer uniquement un fichier géré par HAA après une nouvelle inspection ; --target requis", "Reemplazar solo un archivo administrado por HAA tras una nueva inspección; requiere --target", "Substituir somente um arquivo gerenciado pelo HAA após nova inspeção; requer --target", "HAA 管理を確認したファイルだけを再検査後に置換します。--target が必要です"},
	"Provider update:":     {"Provider-Aktualisierung:", "Mise à jour des Providers :", "Actualización de Providers:", "Atualização de Providers:", "Provider の更新:"},
	"Inspected providers:": {"Geprüfte Provider:", "Providers inspectés :", "Providers inspeccionados:", "Providers inspecionados:", "検査済み Provider 数:"},
	"Inspect and upgrade every required provider within the current version constraints": {"Alle erforderlichen Provider prüfen und innerhalb der aktuellen Versionsvorgaben aktualisieren", "Vérifier et mettre à jour tous les Providers requis selon les contraintes de version actuelles", "Inspeccionar y actualizar todos los Providers requeridos según las restricciones de versión actuales", "Inspecionar e atualizar todos os Providers exigidos dentro das restrições de versão atuais", "現在のバージョン制約に従い、必要なすべての Provider を検査して更新します"},
	"Enroll providerless native or changed managed controls with -upgrade":               {"Native Konfiguration ohne installierte Provider oder geänderte verwaltete Konfiguration mit -upgrade registrieren", "Enregistrer avec -upgrade une configuration native sans Provider installé ou une configuration gérée modifiée", "Registrar con -upgrade una configuración nativa sin Providers instalados o una configuración administrada modificada", "Registrar com -upgrade uma configuração nativa sem Providers instalados ou uma configuração gerenciada alterada", "-upgrade と共に、Provider 未インストールの通常設定または変更した管理設定を登録します"},
	"Update": {"Aktualisierung", "Mise à jour", "Actualización", "Atualização", "更新"},
	"Update an npm project after complete quarantine checks":                         {"npm-Projekt nach vollständiger Quarantäneprüfung aktualisieren", "Mettre à jour un projet npm après les contrôles de quarantaine", "Actualizar un proyecto npm después de la cuarentena completa", "Atualizar um projeto npm após a quarentena completa", "完全な検疫後に npm プロジェクトを更新します"},
	"Explicitly replace supported native project dependencies after full inspection": {"Unterstützte bestehende Projektabhängigkeiten nach vollständiger Prüfung ausdrücklich ersetzen", "Remplacer explicitement les dépendances prises en charge après inspection complète", "Reemplazar explícitamente las dependencias compatibles tras la inspección completa", "Substituir explicitamente as dependências compatíveis após a inspeção completa", "完全な検査後、対応する既存プロジェクトの依存関係を明示的に置き換えます"},
	"Usage:":                  {"Verwendung:", "Utilisation :", "Uso:", "Uso:", "使い方:"},
	"Available Commands:":     {"Verfügbare Befehle:", "Commandes disponibles :", "Comandos disponibles:", "Comandos disponíveis:", "利用可能なコマンド:"},
	"Global Flags:":           {"Globale Optionen:", "Options globales :", "Opciones globales:", "Opções globais:", "共通オプション:"},
	"Flags:":                  {"Optionen:", "Options :", "Opciones:", "Opções:", "オプション:"},
	"Additional help topics:": {"Weitere Hilfethemen:", "Autres rubriques :", "Más temas de ayuda:", "Mais tópicos de ajuda:", "追加ヘルプ:"},
	"Inspect software artifacts before trusted promotion":        {"Software vor der geprüften Installation untersuchen", "Inspecter les logiciels avant leur installation vérifiée", "Inspeccionar software antes de la instalación verificada", "Inspecionar software antes da instalação verificada", "ソフトウェアを検査し、許可されたファイルだけをインストールします"},
	"Inspect and install npm registry packages":                  {"npm-Pakete prüfen und installieren", "Inspecter et installer des paquets npm", "Inspeccionar e instalar paquetes npm", "Inspecionar e instalar pacotes npm", "npm パッケージの検査・インストール"},
	"Inspect an npm package":                                     {"npm-Paket ohne Installation prüfen", "Inspecter un paquet npm sans l’installer", "Inspeccionar un paquete npm sin instalarlo", "Inspecionar um pacote npm sem instalá-lo", "インストールせずに npm パッケージを検査します"},
	"Inspect and install an npm package":                         {"npm-Paket nach der Prüfung installieren", "Inspecter puis installer un paquet npm", "Inspeccionar e instalar un paquete npm", "Inspecionar e instalar um pacote npm", "npm パッケージを検査してインストールします"},
	"Inspect and install PyPI distributions":                     {"PyPI-Pakete prüfen und installieren", "Inspecter et installer des distributions PyPI", "Inspeccionar e instalar distribuciones PyPI", "Inspecionar e instalar distribuições PyPI", "PyPI 配布ファイルの検査・インストール"},
	"Inspect a PyPI distribution":                                {"PyPI-Paket prüfen", "Inspecter une distribution PyPI", "Inspeccionar una distribución PyPI", "Inspecionar uma distribuição PyPI", "PyPI 配布ファイルを検査します"},
	"Inspect and install a PyPI distribution":                    {"PyPI-Paket nach der Prüfung installieren", "Inspecter puis installer une distribution PyPI", "Inspeccionar e instalar una distribución PyPI", "Inspecionar e instalar uma distribuição PyPI", "PyPI 配布ファイルを検査してインストールします"},
	"Inspect and install public GitHub Release assets":           {"Öffentliche GitHub-Release-Dateien prüfen und installieren", "Inspecter et installer des fichiers GitHub Release publics", "Inspeccionar e instalar archivos públicos de GitHub Release", "Inspecionar e instalar arquivos públicos do GitHub Release", "公開 GitHub Release ファイルの検査・インストール"},
	"Inspect a GitHub Release asset":                             {"GitHub-Release-Datei prüfen", "Inspecter un fichier GitHub Release", "Inspeccionar un archivo de GitHub Release", "Inspecionar um arquivo do GitHub Release", "GitHub Release ファイルを検査します"},
	"Inspect and install a GitHub Release asset":                 {"GitHub-Release-Datei prüfen und installieren", "Inspecter puis installer un fichier GitHub Release", "Inspeccionar e instalar un archivo de GitHub Release", "Inspecionar e instalar um arquivo do GitHub Release", "GitHub Release ファイルを検査してインストールします"},
	"Inspect and build public Go Modules":                        {"Öffentliche Go-Module prüfen und bauen", "Inspecter et compiler des modules Go publics", "Inspeccionar y compilar módulos Go públicos", "Inspecionar e compilar módulos Go públicos", "公開 Go モジュールの検査・ビルド"},
	"Inspect and build public crates.io packages":                {"Öffentliche crates.io-Pakete prüfen und bauen", "Inspecter et compiler des paquets crates.io publics", "Inspeccionar y compilar paquetes públicos de crates.io", "Inspecionar e compilar pacotes públicos do crates.io", "公開 crates.io パッケージの検査・ビルド"},
	"Install public Terraform Providers":                         {"Öffentliche Terraform-Provider installieren", "Installer des Providers Terraform publics", "Instalar Providers públicos de Terraform", "Instalar Providers públicos do Terraform", "公開 Terraform Provider のインストール"},
	"Inspect and install Terraform Providers":                    {"Terraform-Provider prüfen und installieren", "Inspecter puis installer des Providers Terraform", "Inspeccionar e instalar Providers de Terraform", "Inspecionar e instalar Providers do Terraform", "Terraform Provider を検査してインストールします"},
	"Check installed release and trusted Host runtime readiness": {"Installation und Laufzeitbereitschaft prüfen", "Vérifier l’installation et l’environnement d’exécution", "Comprobar la instalación y el entorno de ejecución", "Verificar a instalação e o ambiente de execução", "インストールと実行環境の準備状態を確認します"},
	"Manage CLI display settings":                                {"CLI-Anzeige konfigurieren", "Configurer l’affichage du CLI", "Configurar la visualización del CLI", "Configurar a exibição do CLI", "CLI の表示設定"},
	"Show or save CLI language":                                  {"CLI-Sprache anzeigen oder speichern", "Afficher ou enregistrer la langue du CLI", "Mostrar o guardar el idioma del CLI", "Mostrar ou salvar o idioma do CLI", "CLI の表示言語を確認・保存します"},
	"Show help for this command":                                 {"Hilfe zu diesem Befehl anzeigen", "Afficher l’aide de cette commande", "Mostrar la ayuda de este comando", "Mostrar a ajuda deste comando", "このコマンドのヘルプを表示します"},
	"Output: auto, human or json":                                {"Ausgabe: auto, human oder json", "Sortie : auto, human ou json", "Salida: auto, human o json", "Saída: auto, human ou json", "出力形式: auto、human、json"},
	"CLI language: auto, en, ko, de, fr, es, pt-BR or ja":        {"CLI-Sprache: auto, en, ko, de, fr, es, pt-BR oder ja", "Langue du CLI : auto, en, ko, de, fr, es, pt-BR ou ja", "Idioma del CLI: auto, en, ko, de, fr, es, pt-BR o ja", "Idioma do CLI: auto, en, ko, de, fr, es, pt-BR ou ja", "表示言語: auto、en、ko、de、fr、es、pt-BR、ja"},
	"Color: auto, always or never":                               {"Farben: auto, always oder never", "Couleur : auto, always ou never", "Color: auto, always o never", "Cores: auto, always ou never", "色: auto、always、never"},
	"Progress: auto, always or never":                            {"Fortschritt: auto, always oder never", "Progression : auto, always ou never", "Progreso: auto, always o never", "Progresso: auto, always ou never", "進捗表示: auto、always、never"},
	"Show bounded runtime diagnostics":                           {"Begrenzte Laufzeitdiagnosen anzeigen", "Afficher les diagnostics d’exécution limités", "Mostrar diagnósticos de ejecución limitados", "Mostrar diagnósticos de execução limitados", "限定された実行診断情報を表示します"},
	"Installation target path":                                   {"Installationsziel", "Chemin d’installation", "Ruta de instalación", "Caminho de instalação", "インストール先のパス"},
	"Explicit package source":                                    {"Explizite Paketquelle", "Source explicite du paquet", "Origen explícito del paquete", "Origem explícita do pacote", "明示的なパッケージ配布元"},
	"Inspection":                                                 {"Prüfung", "Inspection", "Inspección", "Inspeção", "検査"},
	"Installation":                                               {"Installation", "Installation", "Instalación", "Instalação", "インストール"},
	"Build":                                                      {"Build", "Compilation", "Compilación", "Compilação", "ビルド"},
	"COMPLETED":                                                  {"ABGESCHLOSSEN", "TERMINÉ", "COMPLETADO", "CONCLUÍDO", "完了"},
	"FAILED":                                                     {"FEHLGESCHLAGEN", "ÉCHEC", "FALLÓ", "FALHOU", "失敗"},
	"INCOMPLETE":                                                 {"UNVOLLSTÄNDIG", "INCOMPLET", "INCOMPLETO", "INCOMPLETO", "未完了"},
	"BLOCKED":                                                    {"BLOCKIERT", "BLOQUÉ", "BLOQUEADO", "BLOQUEADO", "ブロック"},
	"REVIEW REQUIRED":                                            {"PRÜFUNG ERFORDERLICH", "EXAMEN REQUIS", "REVISIÓN NECESARIA", "REVISÃO NECESSÁRIA", "確認が必要"},
	"Operation":                                                  {"Vorgang", "Opération", "Operación", "Operação", "操作"},
	"Policy":                                                     {"Richtlinie", "Politique", "Política", "Política", "ポリシー"},
	"Actual installation":                                        {"Tatsächliche Installation", "Installation effective", "Instalación efectiva", "Instalação efetiva", "実際のインストール"},
	"Artifact:":                                                  {"Artefakt:", "Artefact :", "Artefacto:", "Artefato:", "対象:"},
	"Resolved:":                                                  {"Aufgelöst:", "Version résolue :", "Versión resuelta:", "Versão resolvida:", "確定バージョン:"},
	"Target:":                                                    {"Ziel:", "Destination :", "Destino:", "Destino:", "インストール先:"},
	"Inspected artifacts:":                                       {"Geprüfte Pakete:", "Paquets inspectés :", "Paquetes inspeccionados:", "Pacotes inspecionados:", "検査したパッケージ数:"},
	"Checks performed":                                           {"Durchgeführte Prüfungen", "Contrôles effectués", "Comprobaciones realizadas", "Verificações realizadas", "実行した検査"},
	"No completed check results are available.":                  {"Keine abgeschlossenen Prüfergebnisse verfügbar.", "Aucun résultat de contrôle terminé n’est disponible.", "No hay resultados de comprobaciones completadas.", "Não há resultados de verificações concluídas.", "完了した検査結果はありません。"},
	"required":                                                   {"erforderlich", "obligatoire", "obligatorio", "obrigatório", "必須"},
	"optional":                                                   {"optional", "facultatif", "opcional", "opcional", "任意"},
	"Reason:":                                                    {"Grund:", "Motif :", "Motivo:", "Motivo:", "判定理由:"},
	"Error:":                                                     {"Fehler:", "Erreur :", "Error:", "Erro:", "エラー:"},
	"Limitation:":                                                {"Einschränkung:", "Limite :", "Limitación:", "Limitação:", "検査の制限:"},
	"Declared integrity / digest":                                {"Deklarierte Integrität / Digest", "Intégrité déclarée / digest", "Integridad declarada / digest", "Integridade declarada / digest", "宣言された整合性・ハッシュの検証"},
	"Archive / metadata static inspection":                       {"Statische Archiv-/Metadatenprüfung", "Inspection statique de l’archive et des métadonnées", "Inspección estática del archivo y los metadatos", "Inspeção estática do arquivo e dos metadados", "アーカイブ・メタデータの静的検査"},
	"Isolated behavior inspection":                               {"Isolierte Verhaltensprüfung", "Inspection du comportement en environnement isolé", "Inspección de comportamiento en entorno aislado", "Inspeção de comportamento em ambiente isolado", "隔離環境での動作検査"},
	"Preparing operation":                                        {"Laufzeit vorbereiten", "Préparation de l’environnement", "Preparando el entorno", "Preparando o ambiente", "実行環境を準備中"},
	"Resolving exact dependencies":                               {"Exakte Abhängigkeiten auflösen", "Résolution des dépendances exactes", "Resolviendo dependencias exactas", "Resolvendo dependências exatas", "正確な依存関係を確認中"},
	"Downloading artifact":                                       {"Artefakt herunterladen", "Téléchargement du fichier", "Descargando el archivo", "Baixando o arquivo", "ファイルをダウンロード中"},
	"Verifying integrity":                                        {"Integrität prüfen", "Vérification de l’intégrité", "Verificando la integridad", "Verificando a integridade", "整合性を検証中"},
	"Static / sandbox inspection":                                {"Statische / Sandbox-Prüfung", "Inspection statique / sandbox", "Inspección estática / sandbox", "Inspeção estática / sandbox", "静的検査・隔離環境での検査中"},
	"Evaluating policy":                                          {"Richtlinie auswerten", "Évaluation de la politique", "Evaluando la política", "Avaliando a política", "ポリシーを判定中"},
	"Recording manifest / SBOM":                                  {"Manifest / SBOM erstellen", "Création du manifeste / SBOM", "Creando manifiesto / SBOM", "Criando manifesto / SBOM", "マニフェスト・SBOM を作成中"},
	"Staging verified files":                                     {"Geprüfte Dateien vorbereiten", "Préparation des fichiers vérifiés", "Preparando archivos verificados", "Preparando arquivos verificados", "検証済みファイルを準備中"},
	"Installing verified files":                                  {"Geprüfte Dateien installieren", "Installation des fichiers vérifiés", "Instalando archivos verificados", "Instalando arquivos verificados", "検証済みファイルをインストール中"},
	"Building in sandbox":                                        {"In Sandbox bauen", "Compilation dans la sandbox", "Compilando en sandbox", "Compilando em sandbox", "隔離環境でビルド中"},
	"Recording evidence":                                         {"Nachweise speichern", "Enregistrement des preuves", "Guardando evidencias", "Salvando evidências", "検査根拠を記録中"},
	"Environment: ":                                              {"Umgebung: ", "Environnement : ", "Entorno: ", "Ambiente: ", "実行環境: "},
	"Ready":                                                      {"Bereit", "Prêt", "Preparado", "Pronto", "準備完了"},
	"Not fully ready":                                            {"Nicht vollständig bereit", "Pas entièrement prêt", "Preparación incompleta", "Preparação incompleta", "準備未完了"},
	"UNAVAILABLE":                                                {"NICHT VERFÜGBAR", "INDISPONIBLE", "NO DISPONIBLE", "INDISPONÍVEL", "利用不可"},
	"OK":                                                         {"OK", "OK", "OK", "OK", "正常"},
	"Verified release installation":                              {"Verifizierte Release-Installation", "Installation vérifiée de la release", "Instalación verificada de la release", "Instalação verificada da release", "検証済みリリースのインストール"},
	"Trusted runtime":                                            {"Vertrauenswürdige Laufzeit", "Environnement d’exécution de confiance", "Entorno de ejecución de confianza", "Ambiente de execução confiável", "信頼された実行環境"},
	"Isolated behavior observer":                                 {"Beobachter für isoliertes Verhalten", "Observateur de comportement isolé", "Observador de comportamiento aislado", "Observador de comportamento isolado", "隔離動作の観察機能"},
	"Network policy service":                                     {"Netzwerkrichtliniendienst", "Service de politique réseau", "Servicio de política de red", "Serviço de política de rede", "ネットワークポリシーサービス"},
	"Pinned execution images":                                    {"Festgelegte Ausführungs-Images", "Images d’exécution figées", "Imágenes de ejecución fijadas", "Imagens de execução fixadas", "固定された実行イメージ"},
	"Saved language":                                             {"Gespeicherte Sprache", "Langue enregistrée", "Idioma guardado", "Idioma salvo", "保存された言語"},
	"Current language":                                           {"Aktuelle Sprache", "Langue actuelle", "Idioma actual", "Idioma atual", "現在の言語"},
	"Required checks completed and policy allowed this operation.":                                    {"Erforderliche Prüfungen abgeschlossen; die Richtlinie erlaubt diesen Vorgang.", "Les contrôles obligatoires sont terminés ; la politique autorise cette opération.", "Las comprobaciones obligatorias finalizaron y la política permite esta operación.", "As verificações obrigatórias foram concluídas e a política permite esta operação.", "必須検査が完了し、ポリシーが今回の操作を許可しました。"},
	"A required check was not completed; do not treat this as passed.":                                {"Eine erforderliche Prüfung ist unvollständig; dies ist kein bestandenes Ergebnis.", "Un contrôle obligatoire n’est pas terminé ; ce résultat n’est pas une réussite.", "Una comprobación obligatoria no se completó; no es un resultado aprobado.", "Uma verificação obrigatória não foi concluída; o resultado não foi aprovado.", "必須検査が完了していないため、合格とは判断できません。"},
	"Downloaded bytes do not match the expected integrity.":                                           {"Die heruntergeladenen Daten stimmen nicht mit der erwarteten Integrität überein.", "Les données téléchargées ne correspondent pas à l’intégrité attendue.", "Los datos descargados no coinciden con la integridad esperada.", "Os dados baixados não correspondem à integridade esperada.", "ダウンロードしたデータが期待される整合性と一致しません。"},
	"Verified files were installed at the target shown above.":                                        {"Geprüfte Dateien wurden am oben angegebenen Ziel installiert.", "Les fichiers vérifiés ont été installés à la destination indiquée.", "Los archivos verificados se instalaron en el destino indicado.", "Os arquivos verificados foram instalados no destino indicado.", "検証済みファイルを上記のインストール先に配置しました。"},
	"Inspection completed. Inspect does not install files; ALLOW applies to this bounded inspection.": {"Prüfung abgeschlossen. Inspect installiert keine Dateien; ALLOW gilt nur für diese begrenzte Prüfung.", "Inspection terminée. Inspect n’installe aucun fichier ; ALLOW concerne cette inspection limitée.", "Inspección completada. Inspect no instala archivos; ALLOW se aplica a esta inspección limitada.", "Inspeção concluída. Inspect não instala arquivos; ALLOW vale para esta inspeção limitada.", "検査が完了しました。inspect はファイルをインストールせず、ALLOW は今回の検査範囲に適用されます。"},
}

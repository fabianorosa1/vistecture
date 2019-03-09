package web

import (
	"encoding/json"
	"net/http"
	"os"

	"fmt"

	"strings"

	"log"

	"path/filepath"

	"github.com/AOEpeople/vistecture/v2/application"
	"github.com/AOEpeople/vistecture/v2/model/core"
	"github.com/gobuffalo/packr/v2"
)

type (
	ProjectController struct {
		projectDefinitions    *application.ProjectConfig
		projectLoader         *application.ProjectLoader
		definitionsBaseFolder string
	}

	Result struct {
		Name                 string                    `json:"name"`
		AvailableSubViews    []string                  `json:"availableSubViews"`
		ApplicationsByGroup  *core.ApplicationsByGroup `json:"applicationsByGroup"`
		ApplicationsDto      []*ApplicationDto         `json:"applications"`
		StaticDocumentations []string                  `json:"staticDocumentations"`
	}

	ApplicationDto struct {
		*core.Application
		DependenciesGrouped []*core.DependenciesGrouped `json:"dependenciesGrouped"`
	}
)

var (
	fileServerInstance http.Handler
)

func (p *ProjectController) Inject(definitions *application.ProjectConfig, projectLoader *application.ProjectLoader, definitionsBaseFolder string) {
	p.projectDefinitions = definitions
	p.projectLoader = projectLoader
	p.definitionsBaseFolder = definitionsBaseFolder
}

func (p *ProjectController) IndexAction(w http.ResponseWriter, r *http.Request, localTemplateFolder string) {

	handler := initFileServerInstance(localTemplateFolder)
	handler.ServeHTTP(w, r)
}

func initFileServerInstance(localFolder string) http.Handler {
	if fileServerInstance != nil {
		return fileServerInstance
	}
	var fileSystem http.FileSystem
	if localFolder != "" {
		log.Printf("Using filesystem %v templates for serving", localFolder)
		if _, err := os.Stat(localFolder); os.IsNotExist(err) {
			panic(fmt.Sprintf("Cannot start - Folder %v not exitend", localFolder))
		}
		fileSystem = http.Dir(localFolder)
	} else {
		log.Printf("Using templateBox templates for serving")
		fileSystem = packr.New("templateBox", "./template")
	}
	fileServerInstance = http.FileServer(fileSystem)
	return fileServerInstance
}

func (p *ProjectController) DataAction(w http.ResponseWriter, r *http.Request, documentsFolder string) {
	subViewName, _ := r.URL.Query()["subview"]
	project, errors := p.projectLoader.LoadProject(p.projectDefinitions, p.definitionsBaseFolder, strings.Join(subViewName, ""))

	if errors != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"Error":"%v"}`, errors)
		return
	}
	result := Result{
		Name:                project.Name,
		ApplicationsByGroup: project.GetApplicationsRootGroup(),
	}
	for _, subViewConfig := range p.projectDefinitions.SubViewConfig {
		result.AvailableSubViews = append(result.AvailableSubViews, subViewConfig.Name)
	}

	for _, app := range project.Applications {
		result.ApplicationsDto = append(result.ApplicationsDto, &ApplicationDto{
			Application:         app,
			DependenciesGrouped: app.GetDependenciesGrouped(project),
		})
	}
	files, err := getStaticDocuments(documentsFolder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"Error":"`+err.Error()+`"}`)
		return
	}
	result.StaticDocumentations = files

	b, err := json.Marshal(result)
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:63343")
	w.Header().Set("Access-Control-Allow-Origin", "null")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"Error":"`+err.Error()+`"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(b))

}

func getStaticDocuments(folder string) ([]string, error) {
	var result []string
	if folder == "" {
		return result, nil
	}

	files, err := filepath.Glob(strings.TrimRight(folder, "/") + "/*")
	if err != nil {
		return nil, err
	}
	for _, file := range files {

		fileInfo, fileErr := os.Stat(file)
		if fileErr != nil {
			return nil, fileErr
		}

		if fileInfo.IsDir() {
			continue
		}
		path := strings.TrimPrefix(file, folder)
		result = append(result, path)

	}

	return result, err
}

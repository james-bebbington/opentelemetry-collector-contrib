// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Program otelcontribcol extends the OpenTelemetry Collector with
// additional components.
package main

import (
	"log"

	"go.opentelemetry.io/collector/service"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/version"
	"github.com/pkg/errors"
)

func main() {
	factories, err := components()
	if err != nil {
		log.Fatalf("failed to build components: %v", err)
	}

	info := service.ApplicationStartInfo{
		ExeName:  "otelcontribcol",
		LongName: "OpenTelemetry Contrib Collector",
		Version:  version.Version,
		GitHash:  version.GitHash,
	}

	params := service.Parameters{Factories: factories, ApplicationStartInfo: info}
	if err := run(params); err != nil {
		log.Fatal(err)
	}
}

func runInteractive(params service.Parameters) error {
	app, err := service.New(params)
	if err != nil {
		return errors.Wrap(err, "failed to construct the application")
	}

	err = app.Start()
	if err != nil {
		return errors.Wrap(err, "application run finished with error: %v")
	}

	return nil
}

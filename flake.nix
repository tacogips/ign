{

  description = "ign - A template-based code generation CLI tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    divedra = {
      # Keep the CLI pinned before the upstream workflow-runner split that
      # currently references a missing result-finalization module.
      url = "github:tacogips/divedra/c141361";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-unstable.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      divedra,
    }:
    let
      # Single source of truth for version
      version = builtins.replaceStrings [ "\n" ] [ "" ] (builtins.readFile ./internal/build/VERSION);
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        packages = {
          ign = pkgs.buildGoModule {
            pname = "ign";
            inherit version;
            src = ./.;
            vendorHash = "sha256-esmcKwW2KbmEXIukBKUZgN32qovcSiVld41ZqAxN+y4=";
            subPackages = [ "cmd/ign" ];
            ldflags = [
              "-s"
              "-w"
              "-X github.com/tacogips/ign/internal/build.version=${version}"
            ];
            meta = with pkgs.lib; {
              description = "A template-based code generation CLI tool";
              homepage = "https://github.com/tacogips/ign";
              license = licenses.mit;
              maintainers = [ ];
            };
          };

          default = self.packages.${system}.ign;
        };

        apps = {
          ign = {
            type = "app";
            program = "${self.packages.${system}.ign}/bin/ign";
          };

          default = self.apps.${system}.ign;
        };

        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
            go-task
            divedra.packages.${system}.default
          ];

          shellHook = ''
            export GOPATH="$HOME/.cache/go/github.com/tacogips/ign"
            export GOMODCACHE="$HOME/.cache/go/mod"
            mkdir -p "$GOPATH" "$GOMODCACHE"
            echo "Go development environment ready"
            echo "GOPATH: $GOPATH"
            echo "GOMODCACHE: $GOMODCACHE"
            echo "Go version: $(go version)"
            echo "Task version: $(task --version)"
            echo "golangci-lint version: $(golangci-lint --version)"
            echo "Divedra CLI: available"
          '';
        };
      }
    );
}

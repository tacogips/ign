{

  description = "ign - A template-based code generation CLI tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
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
            version = "0.1.0";

            src = ./.;

            vendorHash = "sha256-LBNiQxZlz/hC0RO8fQHGP2WZZ8UKrWiARo5MuChQUtc=";

            subPackages = [ "cmd/ign" ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${self.rev or "dev"}"
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
          ];

          shellHook = ''
            echo "Go development environment ready"
            echo "Go version: $(go version)"
            echo "Task version: $(task --version)"
            echo "golangci-lint version: $(golangci-lint --version)"
          '';
        };
      }
    );
}

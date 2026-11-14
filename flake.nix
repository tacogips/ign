# Using ign with Nix Flakes
#
# This flake provides the `ign` CLI tool for template-based code generation.
#
# ## Building Locally
#
# Build the ign package:
#   nix build .#ign
#
# Run the built binary:
#   ./result/bin/ign --help
#
# Or run directly without building:
#   nix run .#ign -- --help
#
# ## Using from Other Flakes
#
# ### Method 1: Add as a flake input
#
# Add `ign` as an input in your `flake.nix`:
#
# {
#   inputs = {
#     nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
#     ign.url = "github:tacogips/ign";
#   };
#
#   outputs = { self, nixpkgs, ign }:
#     let
#       system = "x86_64-linux";
#       pkgs = import nixpkgs { inherit system; };
#     in
#     {
#       devShells.${system}.default = pkgs.mkShell {
#         buildInputs = [
#           ign.packages.${system}.default
#         ];
#       };
#     };
# }
#
# Then use it in your development shell:
#   nix develop
#   ign --help
#
# ### Method 2: Run directly without adding as input
#
# Run ign from GitHub directly:
#   nix run github:tacogips/ign -- --help
#
# Run a specific command:
#   nix run github:tacogips/ign -- init --help
#
# ### Method 3: Use in NixOS configuration
#
# Add to your NixOS configuration or home-manager:
#
# {
#   inputs = {
#     nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
#     ign.url = "github:tacogips/ign";
#   };
#
#   outputs = { self, nixpkgs, ign }: {
#     nixosConfigurations.your-hostname = nixpkgs.lib.nixosSystem {
#       system = "x86_64-linux";
#       modules = [
#         {
#           environment.systemPackages = [
#             ign.packages.x86_64-linux.default
#           ];
#         }
#       ];
#     };
#   };
# }
#
# ### Method 4: Use specific commit or branch
#
# Use a specific commit:
#   nix run github:tacogips/ign/COMMIT_SHA -- --help
#
# Use a specific branch:
#   nix run github:tacogips/ign/branch-name -- --help
#
# ## Available Flake Outputs
#
# - packages.${system}.ign - The ign CLI package
# - packages.${system}.default - Default package (ign)
# - apps.${system}.ign - The ign CLI app
# - apps.${system}.default - Default app (ign)
# - devShells.${system}.default - Development environment with Go toolchain
#
# ## Development
#
# To contribute to ign development:
#
# Enter development shell:
#   nix develop
#
# This provides:
# - go (Go compiler)
# - gopls (Go language server)
# - gotools (Go development tools)
# - golangci-lint (Go linter)
# - go-task (Task runner)
#
# Build the project:
#   task build
#
# Run tests:
#   task test
#
# ## Notes
#
# - The flake is compatible with all systems supported by flake-utils.lib.eachDefaultSystem
# - The package includes optimized builds with stripped binaries (-s -w ldflags)
# - Version information is embedded during build from git metadata
# - Private repository access requires proper Git SSH configuration

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
            version = "0.1.4";

            src = ./.;

            vendorHash = "sha256-esmcKwW2KbmEXIukBKUZgN32qovcSiVld41ZqAxN+y4=";

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
            export GOPATH="$HOME/.cache/go/tacogips/ign"
            export GOMODCACHE="$HOME/.cache/go/mod"
            mkdir -p "$GOPATH" "$GOMODCACHE"
            echo "Go development environment ready"
            echo "GOPATH: $GOPATH"
            echo "GOMODCACHE: $GOMODCACHE"
            echo "Go version: $(go version)"
            echo "Task version: $(task --version)"
            echo "golangci-lint version: $(golangci-lint --version)"
          '';
        };
      }
    );
}

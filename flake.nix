{
  description = "my zig project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-23.11";
    flake-utils.url = "github:numtide/flake-utils";
    zig-overlay.url = "github:mitchellh/zig-overlay";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      zig-overlay,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        zig-version = "master";
        pkgs = import nixpkgs {
          inherit system;
        };
        zig = zig-overlay.packages.${system}.${zig-version};
      in

      {
        devShells.default = pkgs.mkShell {
          buildInputs = (
            with pkgs;
            [
              just
              zig
            ]
          );

          shellHook = ''
            zig init
          '';
        };
      }
    );
}

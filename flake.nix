{
  description = "my zig project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-23.11";
    flake-utils.url = "github:numtide/flake-utils";
    zig.url = "github:mitchellh/zig-overlay";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      zig,
    }:
    let

      zig-version = "master";
      overlays = [
        (_: prev: {
          zigpkgs = zig.packages.${prev.system};
        })
      ];
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system overlays;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = (
            with pkgs;
            [
              just
              zigpkgs.${zig-version}
            ]
          );

          shellHook = ''
            zig init
          '';
        };
      }
    );
}

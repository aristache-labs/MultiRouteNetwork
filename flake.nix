{
  description = "MultiRouteNetwork: use all your network adapters at once";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (pkgs: rec {
        MultiRouteNetwork = pkgs.callPackage ./nix/package.nix { inherit version; };
        default = MultiRouteNetwork;
      });

      nixosModules = rec {
        MultiRouteNetwork = import ./nix/module.nix self;
        default = MultiRouteNetwork;
      };

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell { packages = [ pkgs.go_1_27 ]; };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt-rfc-style);
    };
}

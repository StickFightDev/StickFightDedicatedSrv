{ sources ? import ./nix/sources.nix, pkgs ? import ./nix { inherit sources; } }:

pkgs.mkShell {
  name = "stickfightserver-shell";

  buildInputs = with pkgs; [
    go
  ];
}

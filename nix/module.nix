self:
{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.programs.MultiRouteNetwork;
in
{
  options.programs.MultiRouteNetwork = {
    enable = lib.mkEnableOption ''
      MultiRouteNetwork, which balances connections across all network adapters. Installs a capability
      wrapper (CAP_NET_ADMIN, CAP_NET_RAW) at /run/wrappers/bin/MultiRouteNetwork so the tray and
      `MultiRouteNetwork run` work without root
    '';

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "MultiRouteNetwork.packages.\${system}.default";
      description = "The MultiRouteNetwork package to use.";
    };

    autostart = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = ''
        Start the MultiRouteNetwork tray in every graphical session, as a systemd user service bound to
        graphical-session.target.
      '';
    };

    extraArgs = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [
        "-ipv6"
        "on"
        "-include"
        "wlp2s0,enp0s20f0u1"
      ];
      description = "Extra flags passed to `MultiRouteNetwork tray` by the autostart service.";
    };
  };

  config = lib.mkIf cfg.enable {
    security.wrappers.MultiRouteNetwork = {
      source = lib.getExe cfg.package;
      owner = "root";
      group = "root";
      # NET_ADMIN creates the TUN and its routes; NET_RAW lets SO_BINDTODEVICE pin flows to an
      # adapter on kernels older than 5.7.
      capabilities = "cap_net_admin,cap_net_raw+ep";
    };

    # Replies to captured flows arrive on the TUN; they belong to connections this machine opened.
    networking.firewall.trustedInterfaces = [ "MultiRouteNetwork" ];

    systemd.user.services.MultiRouteNetwork = lib.mkIf cfg.autostart {
      description = "MultiRouteNetwork tray (use all network adapters at once)";
      partOf = [ "graphical-session.target" ];
      after = [ "graphical-session.target" ];
      wantedBy = [ "graphical-session.target" ];
      serviceConfig = {
        ExecStart = lib.escapeShellArgs (
          [
            "${config.security.wrapperDir}/MultiRouteNetwork"
            "tray"
          ]
          ++ cfg.extraArgs
        );
        Restart = "on-failure";
        RestartSec = 5;
      };
    };
  };
}

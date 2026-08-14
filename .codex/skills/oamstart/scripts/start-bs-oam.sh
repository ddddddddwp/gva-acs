#!/usr/bin/env bash
set -euo pipefail

container="${BS_OAM_CONTAINER:-gva-acs-bs}"
image="${BS_OAM_IMAGE:-bs:5GNR_t.5.1.0.r62694M_20241219_190249}"
runtime_root="${BS_OAM_RUNTIME_ROOT:-/root/code/gva-acs/bs-runtime}"
config_dir="${BS_OAM_CONFIG_DIR:-${runtime_root}/root/hb_ping/BS_config}"
log_dir="${BS_OAM_LOG_DIR:-${runtime_root}/logs}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_config=(m2mcfg.xml packtCtrlCfg.xml simulation_config_bs.txt protStackCfg.sh dataplane_env)

fail() { printf '[FAIL] %s\n' "$*" >&2; }

if ! command -v docker >/dev/null 2>&1; then
  fail "Docker CLI is not installed or not in PATH."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  fail "Docker daemon is unavailable."
  exit 1
fi

if docker container inspect "$container" >/dev/null 2>&1; then
  if [ "$(docker inspect -f '{{.State.Running}}' "$container")" = "true" ]; then
    printf 'Container %s is already running; preserving it unchanged.\n' "$container"
  else
    printf 'Starting existing container %s.\n' "$container"
    docker start "$container" >/dev/null
  fi
else
  if ! docker image inspect "$image" >/dev/null 2>&1; then
    fail "Required image is missing: ${image}"
    exit 1
  fi

  if [ ! -d "$config_dir" ]; then
    fail "Required configuration directory is missing: ${config_dir}"
    exit 1
  fi

  missing=0
  for entry in "${required_config[@]}"; do
    if [ ! -f "${config_dir}/${entry}" ]; then
      fail "Required configuration file is missing: ${config_dir}/${entry}"
      missing=1
    fi
  done
  if (( missing > 0 )); then
    exit 1
  fi

  mkdir -p "$log_dir"
  printf 'Creating BS/OAM container %s from %s.\n' "$container" "$image"
  docker run \
    --name "$container" \
    --hostname "$container" \
    --detach \
    --privileged \
    --restart no \
    --network bridge \
    --add-host host.docker.internal:host-gateway \
    --publish 8400:8400 \
    --publish 7547:7547 \
    --tmpfs /run \
    --tmpfs /run/lock \
    --volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --volume /etc/localtime:/etc/localtime:ro \
    --volume "${config_dir}:/config:ro" \
    --volume "${log_dir}:/opt/bbu/oam/log" \
    --env SIMULATION_IP=192.168.10.212 \
    --env SIMULATION_PORT=65432 \
    --env BSID=12345abc \
    "$image" \
    /bin/bash -lc \
    'cp /config/m2mcfg.xml /home/m2m/m2m-BS/m2mcfg.xml; cp /config/packtCtrlCfg.xml /home/m2m/packet_ctrl/packtCtrlCfg.xml; cp /config/simulation_config_bs.txt /opt/bbu/oam/cm/simulation_config_bs.txt; cp /config/protStackCfg.sh /opt/bbu/oam/cm/protStackCfg.sh; cp /config/dataplane_env /opt/bbu/oam/cm/dataplane_env; chmod 666 /home/m2m/m2m-BS/m2mcfg.xml /home/m2m/packet_ctrl/packtCtrlCfg.xml /opt/bbu/oam/cm/simulation_config_bs.txt /opt/bbu/oam/cm/protStackCfg.sh /opt/bbu/oam/cm/dataplane_env; mkdir -p /run/sshd; ssh-keygen -A; /usr/sbin/sshd; cd /home/m2m/m2m-BS; exec ./start_macadapter_bs.sh' \
    >/dev/null
fi

exec "${script_dir}/status-bs-oam.sh"

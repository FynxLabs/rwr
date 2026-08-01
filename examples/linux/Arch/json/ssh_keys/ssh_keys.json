{
  "ssh_keys": [
    {
      "name": "id_ed25519",
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@personal",
      "no_passphrase": false,
      "copy_to_github": false
    },
    {
      "name": "id_work",
      "profiles": ["work"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@company.com",
      "no_passphrase": false,
      "copy_to_github": false
    },
    {
      "name": "id_github",
      "profiles": ["dev", "github"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@github",
      "no_passphrase": true,
      "copy_to_github": true,
      "github_title": "{{ .User.username }} Development Machine"
    },
    {
      "name": "id_rsa_legacy",
      "profiles": ["legacy", "work"],
      "type": "rsa",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@legacy-systems",
      "no_passphrase": false,
      "copy_to_github": false
    },
    {
      "name": "id_deploy",
      "profiles": ["deploy", "work"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@deployment",
      "no_passphrase": true,
      "copy_to_github": false
    },
    {
      "name": "id_gaming",
      "profiles": ["gaming"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@gaming-rig",
      "no_passphrase": true,
      "copy_to_github": false
    },
    {
      "name": "id_backup",
      "profiles": ["backup", "personal"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@backup-server",
      "no_passphrase": false,
      "copy_to_github": false
    },
    {
      "name": "id_rwr",
      "profiles": ["dev", "work"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@rwr-management",
      "no_passphrase": true,
      "copy_to_github": false,
      "set_as_rwr_ssh_key": true
    },
    {
      "name": "id_cloud",
      "profiles": ["aws", "gcp", "azure", "cloud"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@cloud-services",
      "no_passphrase": false,
      "copy_to_github": false
    },
    {
      "name": "id_docker",
      "profiles": ["docker", "dev"],
      "type": "ed25519",
      "path": "{{ .User.home }}/.ssh/",
      "comment": "{{ .User.username }}@docker-host",
      "no_passphrase": true,
      "copy_to_github": false
    }
  ]
}
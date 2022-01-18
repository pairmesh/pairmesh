<template>
  <el-main style="padding: 0; margin-bottom: 3em;">
    <div class="keys-title">
      <h2>
        <i class="el-icon-document-copy" style="margin-right: 0.5em"></i>
        <span>Keys</span>
      </h2>
      <el-button type="primary" @click="showGenerateKeyPanel = true" plain>
        <i class="el-icon-money" style="margin-right: 0.5em"></i>
        <span>Generate Key</span>
      </el-button>
    </div>

    <el-dialog title="Generate Pre-auth Key" :visible="showGenerateKeyPanel" @close="showGenerateKeyPanel = false">
      <div class="generate-key-panel">
        <div style="margin-bottom: 2em">
          Pre-authentication keys let you register new devices to your user account without an interactive login. These keys are private to you.
        </div>

        <div>
          <el-radio-group v-model="keyType" style="display: flex; flex-direction: column;">
            <el-radio label="one-off" class="key-type-item">
              <span>One-off</span>
              <div style="margin-top: 0.5em">Authenticates a single machine, and then expires.</div>
            </el-radio>
            <el-radio label="reusable" class="key-type-item" style="margin: 1.5em 0">
              <span>Reusable</span>
              <div style="margin-top: 0.5em">Authenticates many machines.</div>
            </el-radio>
          </el-radio-group>
        </div>
        <el-button v-if="!showKey" type="primary" @click="generateKey" style="margin: 1em 0" plain>Generate
        </el-button>

        <el-card v-if="showKey" style="margin-bottom: 1em">
          <h4 style="line-height: 2em; font-size: 1.2em; margin-bottom: 0.5em;">Generated new key</h4>
          <div style="color: #777; font-size: 0.9em">Be sure to copy your new key below. It won't be shown in full
            again.
          </div>
          <div style="margin: 1em 0">
            <el-tag size="medium">{{ this.latestKey }}</el-tag>
          </div>
          <div>
            <el-button @click="showKey=false">Done</el-button>
          </div>
        </el-card>
      </div>
    </el-dialog>

    <el-table :data="keys" style="width: 100%">
      <el-table-column prop="key" label="Key"></el-table-column>
      <el-table-column label="Type">
        <template #default="props">
          <el-tag>{{ props.row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Status">
        <template #default="props">
          <el-tag v-if="props.row.enabled" type="success">Enabled</el-tag>
          <el-tag v-if="!props.row.enabled" type="danger">Disabled</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Created">
        <template #default="props">
          {{ new Date(props.row.created).toLocaleDateString() }}
        </template>
      </el-table-column>
      <el-table-column label="Expiry">
        <template #default="props">
          {{ new Date(props.row.expiry).toLocaleDateString() }}
        </template>
      </el-table-column>
      <el-table-column label="" width="120">
        <template #default="props">
          <el-dropdown trigger="click" @command="handleKey">
            <el-icon class="el-icon-more" style="font-size: 1.3em"></el-icon>
            <el-dropdown-menu>
              <el-dropdown-item icon="el-icon-circle-check"
                                :command="{cmd: 'enable', row: props.row}"
                                :disabled=props.row.enabled>Enable Key
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-circle-close"
                                :command="{cmd: 'disable', row: props.row}"
                                :disabled=!props.row.enabled>Disable Key
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-delete"
                                :command="{cmd: 'delete', row: props.row}">Delete Key
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>
  </el-main>
</template>

<script>
import service from "@/api/service";

export default {
  name: 'keys',
  data: function () {
    return {
      keys: [],
      keyType: 'one-off',
      showKey: false,
      latestKey: '',
      showGenerateKeyPanel: false,
    }
  },
  mounted() {
    this.loadKeys()
  },
  methods: {
    generateKey: function () {
      let self = this
      service.post('/api/v1/key', {
        'type': this.keyType,
      }).then(res => {
        self.latestKey = res.data.key
        self.showKey = true
        res.data.key = res.data.key.slice(0, 12) + "..."
        self.keys.splice(0, 0, res.data)
      })
    },
    loadKeys: function () {
      let self = this
      service.get('/api/v1/keys').then(res => self.keys = res.data.keys)
    },
    handleKey: function (command) {
      if (command.cmd === "enable" || command.cmd === "disable") {
        service.put('/api/v1/key/'+command.row.key_id, {
          'op': command.cmd
        }).then(() => {
            command.row.enabled = command.cmd === "enable"
        })
      }

      if (command.cmd === "delete") {
        let self = this;
        service.delete('/api/v1/key/'+command.row.key_id).then(() => {
          let index = self.keys.indexOf(command.row)
          if (index > -1) {
            self.keys.splice(index, 1);
          }
        })
      }
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

.keys-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.generate-key-panel {
  margin: 0 1em;
}

.generate-key-panel h4 {
  margin: 0;
  line-height: 2.4em;
}
</style>

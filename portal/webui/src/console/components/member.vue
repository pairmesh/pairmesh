<template>
  <el-main style="padding: 0; margin-bottom: 3em;">
    <div class="member-title">
      <div>
        <h2>
          <i class="el-icon-user" style="margin-right: 0.5em"></i>
          <span>Members</span>
        </h2>
      </div>

      <div v-if="admin">
        <el-popover ref="inviteDashboard" placement="bottom" trigger="click" v-model="invitationVisible">
          <el-form :model="invitationForm" :rules="validator">
            <el-form-item prop="email" required>
              <el-input type="text"
                        v-model="invitationForm.email"
                        style="width: 15em"
                        placeholder="EMAIL"
                        autocomplete="off" clearable></el-input>
              <el-select v-model="invitationForm.memberType"
                         style="width: 8em; margin: 0 0.5em">
                <el-option key="member" label="Member" value="member"></el-option>
                <el-option key="admin" label="Admin" value="member"></el-option>
              </el-select>
              <el-button type="primary" plain @click="confirmInviteUser">Invite</el-button>
            </el-form-item>
          </el-form>
        </el-popover>
        <el-button v-popover:inviteDashboard type="primary" plain style="margin-left: 0.5em">
          <i class="el-icon-circle-plus-outline"></i>
          <span>Invite Member</span>
          <i style="margin-left: 0.5em" class="el-icon-arrow-down"></i>
        </el-button>
        <el-button type="danger" plain style="margin-left: 0.5em" @click="handleDeleteNetwork">
          <i class="el-icon-delete"></i>
          <span>Delete Network</span>
        </el-button>
      </div>
    </div>

    <el-table :data="members" style="width: 100%; margin-top: 1em">
      <el-table-column prop="name" label="Name"></el-table-column>
      <el-table-column prop="email" label="Email"></el-table-column>
      <el-table-column label="Role">
        <template #default="props">
          <el-tag :type="props.row.role === 'owner' ? 'warning' : props.row.role === 'member' ? 'success' : ''">
            {{ props.row.role.toUpperCase() }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Join At">
        <template #default="props">
          {{ new Date(props.row.join_time * 1000).toLocaleDateString() }}
        </template>
      </el-table-column>
      <el-table-column width="40" align="right">
        <template #default="props">
          <el-dropdown trigger="click" @command="handleUserOptions">
            <el-icon class="el-icon-more" style="font-size: 1.3em"></el-icon>
            <el-dropdown-menu>
              <el-dropdown-item icon="el-icon-mobile"
                                :command="{cmd: 'showDevices', row: props.row}">Show devices
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-user"
                                :command="{cmd: 'grantAdmin', row: props.row}"
                                :disabled="!owner || props.row.role !== 'member'">Grant admin permission
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-user"
                                :command="{cmd: 'revokeAdmin', row: props.row}"
                                :disabled="!owner || props.row.role !== 'admin'">Revoke admin permission
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-delete"
                                :command="{cmd: 'removeUser', row: props.row}"
                                :disabled="!admin || props.row.role === 'owner'">Remove
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="Devices" :visible="devicesVisible" @close="devicesVisible = false" center>
      <el-table :data="devices" empty-text="NO DEVICES">
        <el-table-column prop="name" label="Name"></el-table-column>
        <el-table-column prop="os" label="OS">
          <template #default="props">
            {{ props.row.os }} ({{ props.row.version }})
          </template>
        </el-table-column>
        <el-table-column prop="address" label='Address'></el-table-column>
        <el-table-column label="Last seen" width="140">
          <template #default="props">
            {{ new Date(props.row.last_seen).toLocaleDateString() }}
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </el-main>
</template>

<script>
import service from "@/api/service";

let checkEmail = (rule, value, callback) => {
  console.log("vailidator")
  const mailReg = /^([a-zA-Z0-9_-])+\.([a-zA-Z0-9_-])*@([a-zA-Z0-9_-])+(.[a-zA-Z0-9_-])+/
  if (!value) {
    return callback(new Error('Email cannot be empty'))
  }
  setTimeout(() => {
    if (mailReg.test(value)) {
      callback()
    } else {
      callback(new Error('Wrong email format'))
    }
  }, 100)
}

export default {
  name: 'member',
  data: function () {
    return {
      members: [],
      admin: false,
      owner: false,
      networkID: 0,
      invitationVisible: false,
      invitationForm: {
        email: '',
        memberType: "Member",
      },
      validator: {
        email: [{validator: checkEmail, trigger: 'blur'}],
      },
      dropdownIndex: 0,
      devicesVisible: false,
      devices: [],
      deleteNetworkVisible: false
    }
  },
  mounted() {
    let self = this;
    self.networkID = this.$route.params.network_id
    service.get("/api/v1/network/" + self.networkID + "/members").then(res => {
      this.members = res.data.members
      this.admin = res.data.admin
      this.owner = res.data.owner
    }).catch(res => {
      self.$message.error(res.data.error)
    })
  },
  methods: {
    confirmInviteUser: function () {
      let self = this;
      service.post('/api/v1/network/' + this.networkID + '/member/invite', {
        'email': this.invitationForm.email,
        'role': this.invitationForm.memberType,
      }).then(() => {
        self.$message.success('Invitation sent successfully');
      }).catch(res => {
        self.$message.error(res.data.error);
      })
      self.invitationVisible = false
    },
    handleUserOptions: function (command) {
      let cmd = command.cmd
      let row = command.row

      if (cmd === 'showDevices') {
        if (row.expanded) {
          this.devices = row.devices
          this.devicesVisible = true
          return
        }

        row.expanded = true
        let self = this;
        service.get('/api/v1/user/' + row.user_id + '/devices')
            .then(res => {
              row.devices = res.data.devices
              self.devices = res.data.devices
              self.devicesVisible = true
            })
      } else if (cmd === 'grantAdmin' || cmd === 'revokeAdmin') {
        service.put('/api/v1/network/' + this.networkID + '/member/' + row.user_id + '/role', {
          'role': cmd === 'grantAdmin' ? 'admin' : 'member',
        }).then(res => {
          row.role = res.data.role
        })
      } else if (cmd === 'removeUser') {
        let self = this
        service.delete('/api/v1/network/' + this.networkID + '/member/' + row.user_id).then(res => {
          let index = self.members.findIndex(el => el.user_id === res.data.user_id)
          if (index > -1) {
            self.members.splice(index, 1)
          }
        })
      }
    },
    handleDeleteNetwork: function () {
      let self = this;
      service.delete('/api/v1/network/' + this.networkID).then(() => {
        self.$message.success('Delete network successfully');
        this.$router.push('/console/networks')
      }).catch(res => {
        self.$message.error(res.data.error);
      })
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

.member-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.el-icon-arrow-down {
  font-size: 1em;
}

</style>

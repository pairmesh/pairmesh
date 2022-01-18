<template>
  <el-main style="padding: 0; margin-bottom: 3em;">
    <div v-if="invitations != null && invitations.length > 0"
         style="display: flex;flex-direction: column;align-items: center;align-content: center;">
      <el-button type="success" plain round size="mini" @click="showInvitations = !showInvitations">
        {{ showInvitations ? 'Hide' : 'Show' }} {{ invitations.length > 1 ? 'invitations' : 'invitation' }}
      </el-button>
      <el-row :gutter="12" style="margin-top: 1em;width: 100%;" v-if="showInvitations">
        <el-col :span="24">
          <el-card shadow="never">
            <el-table :data="invitations" style="width: 100%" :show-header=false>
              <el-table-column>
                <template #default="props">
                  <div style="font-size: 1.1em; color: #444">
                    You’ve been invited to the <span style="color: #3a74fa; font-weight: 500">{{
                      props.row.network_name
                    }}</span> network!
                  </div>
                  <div style="font-size: 0.9em; color: #777">Invited by {{ props.row.invite_user_name }}
                    ({{ props.row.invited_by_user_email }})
                  </div>
                </template>
              </el-table-column>
              <el-table-column width="180" align="right">
                <template #default="props">
                  <el-button @click="handleInvitation(props.row.invitation_id, 'join')" type="success" size="mini"
                             plain>Join
                  </el-button>
                  <el-button @click="handleInvitation(props.row.invitation_id, 'join')" type="danger" size="mini" plain>
                    Decline
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>
    <h2>
      <i class="el-icon-mobile-phone" style="margin-right: 0.5em"></i>
      <span>Devices</span>
    </h2>
    <el-table :data="devices" style="width: 100%" empty-text="NO DEVICES">
      <el-table-column label="Device">
        <template #default="props">
          <div class="device-primary">{{ props.row.name }}</div>
          <div>
            <el-tag v-if="props.row.revoked" :type="'danger'" class="device-tag">revoked</el-tag>
            <el-tag v-if="props.row.updatable" :type="'success'" class="device-tag">updatable</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="OS">
        <template #default="props">
          <div class="device-primary">{{ props.row.os }}</div>
          <div class="device-desc">
            <el-tag class="device-tag">{{ props.row.version }}</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="address" label="Address"></el-table-column>
      <el-table-column label="Last seen" width="140">
        <template #default="props">
          {{ new Date(props.row.last_seen).toLocaleDateString() }}
        </template>
      </el-table-column>
    </el-table>
  </el-main>
</template>

<script>
import service from "@/api/service";

export default {
  name: 'overview',
  data() {
    return {
      devices: [],
      showInvitations: false,
      invitations: [],
    }
  },
  mounted() {
    let self = this;
    service.get("/api/v1/invitations")
        .then(res => self.invitations = res.data.invitations)
    service.get("/api/v1/devices")
        .then(res => self.devices = res.data.devices)
  },
  methods: {
    handleInvitation: function (invitationID, action) {
      let self = this
      service.put("/api/v1/invitation/" + invitationID, {'action': action})
          .then(res => {
            let index = self.invitations.findIndex(el => el.invitation_id === res.data.invitation_id)
            if (index > -1) {
              self.invitations.splice(index, 1)
            }
          })
    },
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
.device-primary {
  font-weight: 500;
  font-size: 1.1em;
}

.device-desc {
  color: #777;
}

.device-tag {
  height: 1.8em;
  line-height: 1.8em;
  padding: 0 0.5em;
  margin-right: 0.5em;
}
</style>

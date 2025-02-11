<template>
  <div class="artboard">
    <header>
      <h1>Members list</h1>
      <router-link :to="{ name: 'projects-projectId-members-add' }">
        <label class="plus">
          <SvgIcon name="plus_square" />
          add new
        </label>
      </router-link>
    </header>

    <Tabs :tabs="tabMenu" />

    <table>
      <thead>
        <tr>
          <th class="w-12"></th>
          <th>Name</th>
          <th class="w-16">Servers</th>
          <th class="w-20">Role</th>
          <th class="w-20">Activity</th>
          <th class="w-8"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(item, index) in data.members" :key="index">
          <td>
            <div class="flex items-center">
              <span class="dot" :class="item.online ? 'bg-green-500' : 'bg-gray-200'"></span>
            </div>
          </td>
          <td>{{ item.user_name }}</td>
          <td>{{ item.servers_count }}</td>
          <td class="flex items-center">
            <Badge :name="RoleUser[item.role]" />
          </td>
          <td>
            <div class="flex items-center">
              <Toggle
                v-model="item.active"
                :id="index"
                @change="changeMemberActive(index, item.active)"
              />
            </div>
          </td>
          <td>
            <router-link
              active-class="current"
              :to="{
                name: 'projects-projectId-members',
                params: {
                  projectId: props.projectId,
                },
              }"
            >
              <SvgIcon name="setting" class="text-gray-700" />
            </router-link>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="artboard-content">
      <Pagination :total="data.total" @selectPage="onSelectPage" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, getCurrentInstance } from "vue";
import { useRoute } from "vue-router";
import { SvgIcon, Pagination, Badge, Toggle, Tabs } from "@/components";
import { showMessage } from "@/utils/message";

import { getMembers, updateMemberStatus } from "@/api/member";
import { UpdateMemberActiveStatus_Request } from "@proto/member/member";
import { RoleUser } from "@proto/user/user";

const { proxy } = getCurrentInstance();
const route = useRoute();
const data: any = ref({});
const props = defineProps({
  projectId: String,
});

// Tabs section
const tabMenu = [
  {
    name: "Members",
    link: { name: "projects-projectId-members" },
  },
  {
    name: "Invites",
    link: { name: "projects-projectId-members-invites" },
  },
];

const getData = async (routeQuery: any) => {
  if (proxy.$authStore.hasUserRole === 3) {
    routeQuery.member_id = proxy.$authStore.hasUserID;
  }
  routeQuery.project_id = props.projectId;
  await getMembers(routeQuery.member_id, routeQuery.project_id, routeQuery).then((res) => {
    data.value = res.data.result;
  });
};

const onSelectPage = (e) => {
  getData(e);
};

onMounted(() => {
  getData(route.query);
});

const changeMemberActive = async (index: number, online: boolean) => {
  const status = !online;
  data.value.members[Number(index)].active = status;

  await updateMemberStatus(<UpdateMemberActiveStatus_Request>{
    owner_id: proxy.$authStore.hasUserID,
    project_id: props.projectId,
    member_id: data.value.members[Number(index)].member_id,
    status: status,
  })
    .then((res) => {
      if (!status) {
        showMessage(res.data.message, "connextWarning");
      } else {
        showMessage(res.data.message);
      }

      proxy.$errorStore.$reset();
    })
    .catch((err) => {
      showMessage(err.response.data.message, "connextError");
    });
};
</script>

<route lang="yaml">
meta:
  requiresAuth: true
</route>

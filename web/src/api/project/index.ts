import { http } from "@/api";
import {
  ListProjects_Request,
  GetProject_Request,
  CreateProject_Request,
  UpdateProject_Request,
  DeleteProject_Request,
} from "@proto/project/project";

enum URL {
  projects = "v1/projects",
}

const getProjects = async (data?: ListProjects_Request, owner_id?: string) =>
  http("GET", URL.projects, {
    params: {
      limit: data.limit,
      offset: data.offset,
      owner_id: owner_id,
    },
  });

const getProject = async (data: GetProject_Request) => http("GET", URL.projects, { params: data });

const postProject = async (data: CreateProject_Request) =>
  http("POST", URL.projects, { data: data });

const updateProject = async (data: UpdateProject_Request) =>
  http("PATCH", URL.projects, { data: data });

const deleteProject = async (data: DeleteProject_Request) =>
  http("DELETE", URL.projects, { params: data });

export { getProjects, getProject, postProject, updateProject, deleteProject };

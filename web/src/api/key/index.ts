import { http } from "@/api";
import {
  ListPublicKeys_Request,
  GetPublicKey_Request,
  CreatePublicKey_Request,
  UpdatePublicKey_Request,
  DeletePublicKey_Request,
} from "@proto/key/key";

enum URL {
  keys = "v1/keys",
}

const getKeys = async (data?: ListPublicKeys_Request, user_id?: string) =>
  http("GET", URL.keys, {
    params: {
      limit: data.limit,
      offset: data.offset,
      user_id: user_id,
    },
  });

const getKey = async (data: GetPublicKey_Request) => http("GET", URL.keys, { params: data });

const postKey = async (data: CreatePublicKey_Request) => http("POST", URL.keys, { data: data });

const updateKey = async (data: UpdatePublicKey_Request) => http("PATCH", URL.keys, { data: data });

const deleteKey = async (data: DeletePublicKey_Request) =>
  http("DELETE", URL.keys, { params: data });

const getNewKey = async () => http("GET", URL.keys + "/generate");

export { getKeys, getKey, postKey, updateKey, deleteKey, getNewKey };

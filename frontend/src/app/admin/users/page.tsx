"use client";

import { useEffect, useState, useCallback } from "react";

import { apiFetch, apiFetchJSON, ApiError } from "@/lib/api";
import { Card, PageHeader, Pagination, StatusBadge, inputClass, primaryButtonClass, LoadingRow } from "@/components/admin-ui";

type User = { id: string; email: string; role: "admin" | "user"; createdAt: string };

const PAGE_SIZE = 30;

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "user">("user");

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editEmail, setEditEmail] = useState("");
  const [editRole, setEditRole] = useState<"admin" | "user">("user");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search });
      const data = await apiFetchJSON<{ users: User[]; total: number }>(`/api/admin/users?${params}`);
      setUsers(data.users ?? []);
      setTotal(data.total);
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => {
    load();
  }, [load]);

  async function createUser() {
    setError(null);
    try {
      await apiFetchJSON("/api/admin/users", {
        method: "POST",
        body: JSON.stringify({ email: newEmail, password: newPassword, role: newRole }),
      });
      setNewEmail("");
      setNewPassword("");
      setNewRole("user");
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create user");
    }
  }

  function startEdit(u: User) {
    setEditingId(u.id);
    setEditEmail(u.email);
    setEditRole(u.role);
  }

  async function saveEdit(id: string) {
    setError(null);
    try {
      await apiFetchJSON(`/api/admin/users/${id}`, {
        method: "PUT",
        body: JSON.stringify({ email: editEmail, role: editRole }),
      });
      setEditingId(null);
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update user");
    }
  }

  async function deleteUser(id: string) {
    if (!confirm("Delete this user?")) return;
    await apiFetch(`/api/admin/users/${id}`, { method: "DELETE" });
    load();
  }

  async function resetPassword(id: string) {
    const password = prompt("New password (at least 8 characters):");
    if (!password) return;
    try {
      await apiFetchJSON(`/api/admin/users/${id}/password`, { method: "POST", body: JSON.stringify({ password }) });
      alert("Password updated.");
    } catch (err) {
      alert(err instanceof ApiError ? err.message : "Could not update password");
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Users" description={`${total} total`} />

      <Card className="flex flex-wrap items-end gap-3 p-4">
        <label className="flex flex-col gap-1 text-xs text-muted">
          Email
          <input value={newEmail} onChange={(e) => setNewEmail(e.target.value)} className={inputClass} />
        </label>
        <label className="flex flex-col gap-1 text-xs text-muted">
          Password
          <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} className={inputClass} />
        </label>
        <label className="flex flex-col gap-1 text-xs text-muted">
          Role
          <select value={newRole} onChange={(e) => setNewRole(e.target.value as "admin" | "user")} className={inputClass}>
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </label>
        <button onClick={createUser} className={primaryButtonClass}>
          Add user
        </button>
      </Card>
      {error && <p className="text-sm text-red-500">{error}</p>}

      <input
        value={search}
        onChange={(e) => {
          setPage(1);
          setSearch(e.target.value);
        }}
        placeholder="Search by email…"
        className={inputClass}
      />

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">Email</th>
                <th className="p-3 font-medium">Role</th>
                <th className="p-3 font-medium">Created</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={4} />
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-6 text-center text-muted">No users.</td>
                </tr>
              ) : (
                users.map((u) => (
                  <tr key={u.id} className="border-t transition-colors hover:bg-surface-muted">
                    {editingId === u.id ? (
                      <>
                        <td className="p-3"><input value={editEmail} onChange={(e) => setEditEmail(e.target.value)} className={inputClass} /></td>
                        <td className="p-3">
                          <select value={editRole} onChange={(e) => setEditRole(e.target.value as "admin" | "user")} className={inputClass}>
                            <option value="user">user</option>
                            <option value="admin">admin</option>
                          </select>
                        </td>
                        <td className="p-3 text-muted">{new Date(u.createdAt).toLocaleDateString("en-US")}</td>
                        <td className="p-3 flex gap-3">
                          <button onClick={() => saveEdit(u.id)} className="font-medium text-green-600 hover:underline">Save</button>
                          <button onClick={() => setEditingId(null)} className="text-muted hover:underline">Cancel</button>
                        </td>
                      </>
                    ) : (
                      <>
                        <td className="p-3">{u.email}</td>
                        <td className="p-3"><StatusBadge value={u.role} /></td>
                        <td className="p-3 text-muted">{new Date(u.createdAt).toLocaleDateString("en-US")}</td>
                        <td className="p-3 flex gap-3">
                          <button onClick={() => startEdit(u)} className="hover:underline">Edit</button>
                          <button onClick={() => resetPassword(u.id)} className="hover:underline">Change password</button>
                          <button onClick={() => deleteUser(u.id)} className="text-red-500 hover:underline">Delete</button>
                        </td>
                      </>
                    )}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>
      <Pagination page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </div>
  );
}

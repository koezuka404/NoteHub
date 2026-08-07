import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import DocumentEditorPage from './DocumentEditorPage';
import DocumentVersionDetailPage from './DocumentVersionDetailPage';
import DocumentVersionsPage from './DocumentVersionsPage';
import DocumentsPage from './DocumentsPage';
import LoginPage from './LoginPage';
import MembersPage from './MembersPage';
import ProtectedRoute from './ProtectedRoute';
import PublicRoute from './PublicRoute';
import TopPage from './TopPage';
import WorkspacesPage from './WorkspacesPage';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<PublicRoute />}>
          <Route path="/login" element={<LoginPage />} />
        </Route>

        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<TopPage />} />
          <Route path="/workspaces" element={<WorkspacesPage />} />
          <Route path="/workspaces/:workspaceId/documents" element={<DocumentsPage />} />
          <Route path="/workspaces/:workspaceId/members" element={<MembersPage />} />
          <Route path="/workspaces/:workspaceId/documents/:documentId" element={<DocumentEditorPage />} />
          <Route path="/workspaces/:workspaceId/documents/:documentId/versions" element={<DocumentVersionsPage />} />
          <Route
            path="/workspaces/:workspaceId/documents/:documentId/versions/:versionId"
            element={<DocumentVersionDetailPage />}
          />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

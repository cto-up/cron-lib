/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type JobAuditLog = {
    id: string;
    app_id: string;
    request_id: string;
    job_name: string;
    scheduled_time: string;
    start_time?: string;
    end_time?: string;
    status: string;
    output?: string;
    error?: string;
    tenantID: string;
    createdAt: string;
    updatedAt?: string;
};


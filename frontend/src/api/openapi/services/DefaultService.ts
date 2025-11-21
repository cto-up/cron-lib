/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { Job } from '../models/Job';
import type { JobAuditLog } from '../models/JobAuditLog';
import type { RegisteredJob } from '../models/RegisteredJob';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class DefaultService {
    /**
     * Returns all Job audit logs from the system that the user has access to
     *
     * @param page page number
     * @param pageSize maximum number of results to return
     * @param sortBy field to sort by
     * @param order sort order
     * @param q starts with
     * @param detail basic or full
     * @returns JobAuditLog job_audit_log response
     * @throws ApiError
     */
    public static listJobAuditLogs(
        page: number = 1,
        pageSize: number = 10,
        sortBy?: string,
        order?: 'asc' | 'desc',
        q?: string,
        detail?: string,
    ): CancelablePromise<Array<JobAuditLog>> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/job-audit-logs',
            query: {
                'page': page,
                'pageSize': pageSize,
                'sortBy': sortBy,
                'order': order,
                'q': q,
                'detail': detail,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
            },
        });
    }
    /**
     * Returns a job audit log based on a single ID, if the user does not have access to the job audit log
     * @param id ID of job audit log to fetch
     * @returns JobAuditLog job audit log response
     * @throws ApiError
     */
    public static getJobAuditLogById(
        id: string,
    ): CancelablePromise<JobAuditLog> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/job-audit-logs/{id}',
            path: {
                'id': id,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                404: `job audit log not found`,
            },
        });
    }
    /**
     * deletes a single job audit log based on the ID supplied
     * @param id ID of job audit log to delete
     * @returns void
     * @throws ApiError
     */
    public static deleteJobAuditLog(
        id: string,
    ): CancelablePromise<void> {
        return __request(OpenAPI, {
            method: 'DELETE',
            url: '/api/v1/cron/job-audit-logs/{id}',
            path: {
                'id': id,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                404: `job audit log not found`,
            },
        });
    }
    /**
     * Returns all Jobs from the system that the user has access to
     *
     * @param page page number
     * @param pageSize maximum number of results to return
     * @param sortBy field to sort by
     * @param order sort order
     * @param q starts with
     * @param detail basic or full
     * @param lang
     * @returns Job scheduled_task response
     * @throws ApiError
     */
    public static listJobs(
        page: number = 1,
        pageSize: number = 10,
        sortBy?: string,
        order?: 'asc' | 'desc',
        q?: string,
        detail?: string,
        lang: 'en' | 'fr' = 'en',
    ): CancelablePromise<Array<Job>> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/jobs',
            query: {
                'page': page,
                'pageSize': pageSize,
                'sortBy': sortBy,
                'order': order,
                'q': q,
                'detail': detail,
                'lang': lang,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
            },
        });
    }
    /**
     * Returns a Job based on a single ID, if the user does not have access to the Job
     * @param id ID of Job to fetch
     * @param lang
     * @returns Job Job response
     * @throws ApiError
     */
    public static getJobById(
        id: string,
        lang: 'en' | 'fr' = 'en',
    ): CancelablePromise<Job> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/jobs/{id}',
            path: {
                'id': id,
            },
            query: {
                'lang': lang,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                404: `Job not found`,
            },
        });
    }
    /**
     * deletes a single Job based on the ID supplied
     * @param id ID of Job to delete
     * @returns void
     * @throws ApiError
     */
    public static deleteJob(
        id: string,
    ): CancelablePromise<void> {
        return __request(OpenAPI, {
            method: 'DELETE',
            url: '/api/v1/cron/jobs/{id}',
            path: {
                'id': id,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                404: `Job not found`,
            },
        });
    }
    /**
     * List all registered jobs
     * @param page Page number for pagination
     * @param pageSize Maximum number of results to return
     * @param sortBy Field to sort by
     * @param order Sort order
     * @param q Search term for job name
     * @returns RegisteredJob List of registered jobs
     * @throws ApiError
     */
    public static listRegisteredJobs(
        page: number = 1,
        pageSize: number = 10,
        sortBy: 'job_name' | 'schedule' | 'last_registered_at' | 'is_enabled' = 'job_name',
        order: 'asc' | 'desc' = 'asc',
        q?: string,
    ): CancelablePromise<Array<RegisteredJob>> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/registered-jobs',
            query: {
                'page': page,
                'pageSize': pageSize,
                'sortBy': sortBy,
                'order': order,
                'q': q,
            },
            errors: {
                400: `Bad request`,
                401: `Unauthorized`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Get a registered job by ID
     * @param id ID of registered job to fetch
     * @returns RegisteredJob Registered job details
     * @throws ApiError
     */
    public static getRegisteredJob(
        id: string,
    ): CancelablePromise<RegisteredJob> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/registered-jobs/{id}',
            path: {
                'id': id,
            },
            errors: {
                401: `Unauthorized`,
                404: `Job not found`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Update a registered job (currently only enables/disables)
     * @param id ID of registered job to update
     * @param requestBody
     * @returns RegisteredJob Updated job details
     * @throws ApiError
     */
    public static updateRegisteredJob(
        id: string,
        requestBody: {
            /**
             * Whether the job is enabled
             */
            is_enabled?: boolean;
        },
    ): CancelablePromise<RegisteredJob> {
        return __request(OpenAPI, {
            method: 'PATCH',
            url: '/api/v1/cron/registered-jobs/{id}',
            path: {
                'id': id,
            },
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Bad request`,
                401: `Unauthorized`,
                404: `Job not found`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Get audit logs for a specific registered job
     * @param id ID of registered job to fetch
     * @param page page number
     * @param pageSize maximum number of results to return
     * @param sortBy field to sort by
     * @param order sort order
     * @param q starts with
     * @returns JobAuditLog job_audit_log response
     * @throws ApiError
     */
    public static getJobAuditLogs(
        id: string,
        page: number = 1,
        pageSize: number = 10,
        sortBy?: string,
        order?: 'asc' | 'desc',
        q?: string,
    ): CancelablePromise<Array<JobAuditLog>> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/cron/registered-jobs/{id}/audit-logs',
            path: {
                'id': id,
            },
            query: {
                'page': page,
                'pageSize': pageSize,
                'sortBy': sortBy,
                'order': order,
                'q': q,
            },
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
            },
        });
    }
    /**
     * Apply pending migrations
     * @returns any Migrations applied successfully
     * @throws ApiError
     */
    public static migrateUp(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/cron/migrate/up',
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Revert last migration
     * @returns any Migration reverted successfully
     * @throws ApiError
     */
    public static migrateDown(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/cron/migrate/down',
            errors: {
                401: `Unauthorized`,
                403: `Forbidden`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Seeds reference data for module
     * @returns any Reference data seeded successfully
     * @throws ApiError
     */
    public static seedReferenceData(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/cron/seed/reference',
            errors: {
                401: `Unauthorized`,
                403: `Forbidden - requires admin privileges`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Seeds sample data for module
     * @returns any Sample data seeded successfully
     * @throws ApiError
     */
    public static seedSampleData(): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/cron/seed/sample',
            errors: {
                401: `Unauthorized`,
                403: `Forbidden - requires admin privileges`,
                500: `Internal server error`,
            },
        });
    }
}

import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {JobExecutionStatusListComponent} from './job-execution-status-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, JobExecutionStatus, ListDataSource, Meta} from '../../../../shared/models';
import {JobExecutionService} from '../../services';

describe('JobExecutionStatusListComponent', () => {
  let component: JobExecutionStatusListComponent;
  let fixture: ComponentFixture<JobExecutionStatusListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [JobExecutionStatusListComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: JobExecutionService,
          useValue: {
            getJob: () => of(new ConfigObject({meta: new Meta({name: 'Daily crawl'})})),
          },
        },
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the job name as plain row content and keeps row navigation', async () => {
    const row = new JobExecutionStatus({id: 'execution-1', jobId: 'job-1'});
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    const clicked: JobExecutionStatus[] = [];
    component.rowClick.subscribe(item => clicked.push(item));

    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const itemRow = fixture.nativeElement.querySelector('.item-row') as HTMLElement;
    const jobCell = itemRow.querySelector('td') as HTMLElement;
    expect(jobCell.textContent).toContain('Daily crawl');
    expect(jobCell.querySelector('a')).toBeNull();

    itemRow.click();
    itemRow.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(clicked).toEqual([row, row]);
  });
});

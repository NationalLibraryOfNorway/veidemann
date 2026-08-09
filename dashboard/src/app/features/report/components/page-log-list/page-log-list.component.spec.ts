import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogListComponent} from './page-log-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {of} from 'rxjs';
import {ListDataSource, PageLog} from '../../../../shared/models';


describe('PageLogListComponent', () => {
  let component: PageLogListComponent;
  let fixture: ComponentFixture<PageLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        PageLogListComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('links the displayed URI externally while leaving the row as the detail target', async () => {
    const row = new PageLog({warcId: 'warc-1', uri: 'https://example.org/page'});
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const links = fixture.nativeElement.querySelectorAll('td a') as NodeListOf<HTMLAnchorElement>;
    expect(component.displayedColumns).toEqual(['uri', 'nrOfResources', 'nrOfOutlinks']);
    expect(links.length).toBe(1);
    expect(links[0].href).toBe('https://example.org/page');
    expect(links[0].target).toBe('_blank');
    expect(fixture.nativeElement.querySelector('.action-cell')).toBeNull();
  });
});
